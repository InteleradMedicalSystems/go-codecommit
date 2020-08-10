package codecommit

import (
	"fmt"
	nurl "net/url"
	"os"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
)

const (
	envKeyAwsAccessKeyID     = "AWS_ACCESS_KEY_ID"
	envKeyAwsSecretAccessKey = "AWS_SECRET_ACCESS_KEY"
)

//
var RegionRe *regexp.Regexp

func init() {
	RegionRe = regexp.MustCompile(`git-codecommit\.([^.]+)\.amazonaws\.com`)
}

//isCodeCommitURL return true if the url is for a CodeCommit Git repo.
func (c *CloneURL) isCodeCommitURL() bool {
	return RegionRe.MatchString(c.u.Host)
}

//NewCloneURL return CloneURL object for CodeCommit. If the URL represents a codecommit URL, the
//aws credentials will be set.
func NewCloneURL(roleArn *string, rawURL string) (*CloneURL, error) {
	url, err := nurl.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	cloneUrl := &CloneURL{
		u: url,
	}

	if cloneUrl.isCodeCommitURL() {
		if err := cloneUrl.setAwsCredentials(roleArn); err != nil {
			return nil, err
		}
	}
	return cloneUrl, nil
}

type CloneURL struct {
	credValues credentials.Value
	u          *nurl.URL
}

//
func (c *CloneURL) buildCloneURL() error {
	if c.u.User == nil && c.isCodeCommitURL() {
		if err := c.addCodeCommitCreds(); err != nil {
			return err
		}
	}
	return nil
}

// Include the CodeCommit HTTP AUTH params .
func (c *CloneURL) addCodeCommitCreds() error {
	if creds, err := c.GetCodeCommitCredentials(); err == nil {
		c.u.User = nurl.UserPassword(creds.Username, creds.Password)
	} else {
		return err
	}
	return nil
}

//String gets the url appropriate for cloning. If the the URL is a codecommit URL, the username and
//password will be added to the url.
func (c *CloneURL) String() string {
	err := c.buildCloneURL()
	if err != nil {
		return ""
	}
	return c.u.String()
}

//GetCodeCommitCredentials return CodeCommitCredentials for URL
func (c *CloneURL) GetCodeCommitCredentials() (*CodeCommitCredentials, error) {
	region, err := c.parseRegion()
	if err != nil {
		return nil, err
	}

	ctx := NewSigningContext(c.u, region, endpoints.CodecommitServiceID, c.credValues, time.Now())
	username := c.credValues.AccessKeyID
	if c.credValues.SessionToken != "" {
		username = fmt.Sprintf("%s%%%s", username, c.credValues.SessionToken)
	}

	return &CodeCommitCredentials{
		Username: username,
		Password: ctx.signCodeCommitRequest(),
	}, nil
}

func (c *CloneURL) parseRegion() (string, error) {
	if c.u == nil {
		return "", fmt.Errorf("url is not set")

	}
	match := RegionRe.FindStringSubmatch(c.u.Host)
	if match != nil {
		return match[1], nil
	}

	return "", fmt.Errorf("invalid CodeCommit URL %q", c.u.String())
}

func (c *CloneURL) setAwsCredentials(roleArn *string) error {
	var creds credentials.Value
	if roleArn != nil {
		if err := validateAssumeRoleConfig(); err != nil {
			return err
		}
		region, err := c.parseRegion()
		if err != nil {
			return err
		}

		cfg := &aws.Config{
			Region: &region,
		}
		sess := session.Must(session.NewSession(cfg))
		if creds, err = stscreds.NewCredentials(sess, *roleArn).Get(); err != nil {
			return err
		}
	} else {
		// Get the creds without sts assume role. AWS_PROFILE and AWS_SDK_LOAD_CONFIG must be set.
		sess, err := session.NewSession()
		if err != nil {
			return err
		}
		if creds, err = sess.Config.Credentials.Get(); err != nil {
			return err
		}
	}
	c.credValues = creds
	return nil
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

type CodeCommitCredentials struct {
	Username string
	Password string
}
