package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bashims/go-codecommit/pkg/codecommit"
)

//GitCmd for commandline execution
type GitCmd struct {
	wrapper codecommit.RepoWrapper
	sess    *session.Session
	region  *string
	roleARN *string
}

func (g *GitCmd) execute(cmd *cobra.Command, args []string) error {
	switch command := cmd.Name(); command {
	case "clone":
		return g.clone(args, cmd.Flags())
	case "pull":
		return g.pull(args)
	case "push":
		return g.push(args)
	default:
		return fmt.Errorf("unsupported command %q", command)
	}
}

func (g *GitCmd) pull(args []string) error {
	var path string
	if len(args) == 1 {
		path = args[0]
	}

	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	err = g.wrapper.Pull(path)
	return err
}

func (g *GitCmd) push(args []string) error {
	var path string
	if len(args) == 1 {
		path = args[0]
	}

	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	err = g.wrapper.Push(path)
	return err
}

func (g *GitCmd) clone(args []string, flags *pflag.FlagSet) error {
	url := os.Getenv(envKeyCodeCommitURL)
	if url == "" {
		if len(args) < 1 {
			return fmt.Errorf("clone URL not provided")
		}
		url, args = args[0], args[1:]
	}

	if codecommit.IsCodeCommitURL(url) {
		roleARN, err := flags.GetString("role-arn")
		if err != nil {
			return err
		}
		if roleARN != "" && os.Getenv(envKeyAwsProfile) != "" {
			return fmt.Errorf("only one of role arn or profile should be set")
		}
		if roleARN != "" {
			g.roleARN = &roleARN
		}

		region, err := codecommit.ParseRegion(url)
		g.region = &region

		sess, err := g.session()
		if err != nil {
			return err
		}
		cloneURL, err := codecommit.NewCloneURL(sess, url)
		if err != nil {
			return err
		}

		url = cloneURL.String()
	}

	var dest string
	if len(args) == 1 {
		dest = args[0]
	} else {
		dest = g.wrapper.GetDestPath(url)
	}

	dest, err := filepath.Abs(dest)
	if err != nil {
		return err
	}

	contents, err := ioutil.ReadDir(dest)
	if err == nil && len(contents) > 0 {
		return fmt.Errorf("%q is not empty, refusing to clone %s", dest, url)
	}

	fmt.Printf("cloning %s to %s\n", url, dest)
	_, _, err = g.wrapper.Clone(url, dest)
	return err
}

//session getter/setter returns *session.session
func (g *GitCmd) session() (*session.Session, error) {
	if g.sess == nil {
		cfg := &aws.Config{
			Region: g.region,
		}
		sess, err := session.NewSession(cfg)
		if err != nil {
			return nil, err
		}

		if g.roleARN != nil {
			sess.Config.Credentials = stscreds.NewCredentials(sess, *g.roleARN)
		}
		g.sess = sess
	}
	return g.sess, nil
}

func newCloneCmd() *cobra.Command {
	c := &GitCmd{}
	cmd := &cobra.Command{
		Use:   "clone URL [directory]",
		Short: "Clone the CodeCommit repository to directory",
		Long: `Git clone a CodeCommit repository.

See: %s for more details

Example usage:

codecommit clone https://git-codecommit.us-east-1.amazonaws.com/v1/repos/your-repo .
`,
		RunE: c.execute,
		Args: cobra.MaximumNArgs(2),
	}

	cmd.Flags().String("role-arn", os.Getenv(envKeyCodeCommitRoleARN), "role to assume when retrieving aws credentials, requires 'AWS_ACCESS_KEY_ID' and 'AWS_SECRET_KEY_ID' env vars to be set")
	return cmd
}

func newPullCmd() *cobra.Command {
	c := &GitCmd{}
	cmd := &cobra.Command{
		Use:   "pull [directory]",
		Short: "Pull updates from the CodeCommit",
		Long: `Git pull a CodeCommit repository.

See: %s for more details

Example usage:

cd your-repo && codecommit pull

Or:

codecommit pull ./your-repo

`,
		RunE: c.execute,
		Args: cobra.MaximumNArgs(1),
	}
	return cmd
}

func newPushCmd() *cobra.Command {
	c := &GitCmd{}
	cmd := &cobra.Command{
		Use:   "push [directory]",
		Short: "Push updates from the CodeCommit",
		Long: `Git push a CodeCommit repository.

See: %s for more details

Example usage:

cd your-repo && codecommit push

Or:

codecommit push ./your-repo
`,
		RunE: c.execute,
		Args: cobra.MaximumNArgs(1),
	}
	return cmd
}
