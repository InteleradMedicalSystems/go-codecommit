package main

import (
	"bufio"
	"fmt"
	nurl "net/url"
	"os"
	"regexp"
	"text/template"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/bashims/go-codecommit/pkg/codecommit"
	"github.com/spf13/cobra"
)

const (
	envKeyCodeCommitURL = "CODECOMMIT_URL"
	helperTemplate      = `username={{ .Credentials.Username }}
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
	sess   *session.Session
	method string
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
		sess, err := session.NewSession()
		if err != nil {
			return nil, err
		}
		c.sess = sess
	}
	return c.sess, nil
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

	return c.emitCreds(url, format)
}

func (c *CodeCommitCredentials) executeCredentialHelper(cmd *cobra.Command, args []string) error {
	r := c.parseGitInput()
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
