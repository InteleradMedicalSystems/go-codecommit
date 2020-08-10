package main

import (
	"bufio"
	"fmt"
	nurl "net/url"
	"os"
	"regexp"
	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	stscreds "github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/spf13/cobra"

	"github.com/bashims/go-codecommit/pkg/codecommit"
)

const (
	envKeyCodeCommitURL = "CODECOMMIT_URL"

	envKeyAwsAccessKeyID     = "AWS_ACCESS_KEY_ID"
	envKeyAwsSecretAccessKey = "AWS_SECRET_ACCESS_KEY"
	envKeyCodeCommitRoleArn  = "GO_CODECOMMIT_ROLE_ARN"

	helperTemplate = `username={{ .Credentials.Username }}
password={{ .Credentials.Password }}
`
	gitCredentialsHelperAPIDoc = "https://git-scm.com/docs/api-credentials#_credential_helpers"
)

//GitRequest contains elements from a parsed Git helper input
//See https://git-scm.com/docs/api-credentials#_credential_helpers
type GitRequest struct {
	protocol string
	path     string
	host     string
}

func (g *GitRequest) url() string {
	u := nurl.URL{
		Host:   g.host,
		Path:   g.path,
		Scheme: g.protocol,
	}
	return u.String()
}

//Values for use in Templating
type Values struct {
	Credentials *codecommit.CodeCommitCredentials
}

type CodeCommitCredentials struct {
	sess    *session.Session
	roleArn *string
	region  *string
	method  string
}

func (c *CodeCommitCredentials) parseGitInput() GitRequest {
	gitInputRe := regexp.MustCompile(`^(.+)=(.+)$`)
	scanner := bufio.NewScanner(os.Stdin)
	r := GitRequest{}
	for scanner.Scan() {
		match := gitInputRe.FindStringSubmatch(scanner.Text())
		if match != nil {
			switch match[1] {
			case "protocol":
				r.protocol = match[2]
			case "host":
				r.host = match[2]
			case "path":
				r.path = match[2]
			}
		}
	}
	return r
}

func (c *CodeCommitCredentials) getCreds(url string) (*codecommit.CodeCommitCredentials, error) {
	cloneURL, err := c.cloneURL(url)
	if err != nil {
		return nil, err
	}

	creds, err := cloneURL.GetCodeCommitCredentials()
	if err != nil {
		return nil, err
	}
	return creds, nil
}

//cloneURL return a codecommit.cloneURL for url
func (c *CodeCommitCredentials) cloneURL(url string) (*codecommit.CloneURL, error) {
	sess, err := c.session()
	if err != nil {
		return nil, err
	}

	cloneURL, err := codecommit.NewCloneURL(sess, url)
	if err != nil {
		return nil, err
	}
	return cloneURL, nil

}

func (c *CodeCommitCredentials) emitCreds(url, format string) error {
	creds, err := c.getCreds(url)
	if err != nil {
		return err
	}

	t := template.Must(template.New("format").Parse(format))

	values := Values{
		Credentials: creds,
	}
	err = t.Execute(os.Stdout, values)
	if err != nil {
		return err
	}
	return nil
}

//session getter/setter returns *session.session
func (c *CodeCommitCredentials) session() (*session.Session, error) {
	if c.sess == nil {
		cfg := &aws.Config{
			Region: c.region,
		}
		sess, err := session.NewSession(cfg)
		if err != nil {
			return nil, err
		}

		if c.roleArn != nil {
			if err := validateAssumeRoleConfig(); err != nil {
				return nil, err
			}
			sess.Config.Credentials = stscreds.NewCredentials(sess, *c.roleArn)
		}
		c.sess = sess
	}
	return c.sess, nil
}

func validateAssumeRoleConfig() error {
	if _, isset := os.LookupEnv(envKeyAwsAccessKeyID); !isset {
		return fmt.Errorf("cannot assume role since the env var: '%s' must be set", envKeyAwsAccessKeyID)
	}
	if _, isset := os.LookupEnv(envKeyAwsSecretAccessKey); !isset {
		return fmt.Errorf("cannot assume role since the env var: '%s' must be set", envKeyAwsSecretAccessKey)
	}
	return nil
}

func (c *CodeCommitCredentials) execute(cmd *cobra.Command, args []string) error {
	f := cmd.Flags()

	url, err := f.GetString("url")
	if err != nil {
		return err
	}
	if url == "" {
		return fmt.Errorf("URL not specified")
	}

	format, err := f.GetString("template")
	if err != nil {
		return err
	}
	if format == "" {
		format = helperTemplate
	}

	roleArn, err := f.GetString("role-arn")
	if err != nil {
		return err
	}
	if roleArn != "" && os.Getenv(envKeyAwsProfile) != "" {
		return fmt.Errorf("only one of role arn or profile should be set")
	}
	if roleArn != "" {
		c.roleArn = &roleArn
	}
	region, err := codecommit.ParseRegion(url)
	if err != nil {
		return err
	}
	c.region = &region

	return c.emitCreds(url, format)
}

func (c *CodeCommitCredentials) executeCredentialHelper(cmd *cobra.Command, args []string) error {
	r := c.parseGitInput()

	if codecommit.IsCodeCommitURL(r.url()) {
		if r, isset := os.LookupEnv(envKeyCodeCommitRoleArn); isset {
			if os.Getenv(envKeyAwsProfile) != "" {
				return fmt.Errorf("only one of role arn or profile should be set")
			}
			c.roleArn = &r
		}

		region, err := codecommit.ParseRegion(r.host)
		if err != nil {
			return err
		}
		c.region = &region
	}

	return c.emitCreds(r.url(), helperTemplate)
}

func newCredentialsCmd() *cobra.Command {
	c := &CodeCommitCredentials{}
	cmd := &cobra.Command{
		Use:   "credential [options]",
		Short: "Emit credentials for URL for method",
		Long: fmt.Sprintf(`Emit CodeCommit credentials

The CodeCommit URL can alternately be set from the environment variable %q.

Output can be templated using standard Go templating on the Credentials object

Templating example(s):
For standard Git credential helper output (the default)
codecommit credential --url https://git-codecommit.us-east-1.amazonaws.com/v1/repos/your-repo \
--template '%s'
`, envKeyCodeCommitURL, helperTemplate),
		RunE: c.execute,
		Args: cobra.ExactArgs(0),
	}

	cmd.Flags().String("url", os.Getenv(envKeyCodeCommitURL),
		fmt.Sprintf("emit credentials for URL\nCan be set from the environment with %s",
			envKeyCodeCommitURL))
	cmd.Flags().String("template", "", "template output (Go templating)")
	cmd.Flags().String("role-arn", os.Getenv(envKeyCodeCommitRoleArn), "role to assume when retrieving aws credentials, requires 'AWS_ACCESS_KEY_ID' and 'AWS_SECRET_KEY_ID' env vars to be set")
	return cmd
}

func newCredentialHelperCmd() *cobra.Command {
	c := &CodeCommitCredentials{}
	cmd := &cobra.Command{
		Use:   "credential-helper [options]",
		Short: "Emit credentials for Git's credential-helper API",
		Long: fmt.Sprintf(`Emit credentials for Git's credential-helper API

See: %s for more details

Example usage:

git clone --config=credential.helper='!codecommit credential-helper $@' \
  --config=credential.UseHttpPath=true \
   https://git-codecommit.us-east-1.amazonaws.com/v1/repos/your-repo .
`, gitCredentialsHelperAPIDoc),
		RunE: c.executeCredentialHelper,
		Args: cobra.ExactArgs(1),
	}
	return cmd
}
