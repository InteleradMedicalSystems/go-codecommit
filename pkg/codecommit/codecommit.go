package codecommit

import (
	"fmt"
	nurl "net/url"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
)

//
var RegionRe *regexp.Regexp

func init() {
	RegionRe = regexp.MustCompile(`git-codecommit\.([^.]+)\.amazonaws\.com`)
}

//IsCodeCommitURL return true if the url is for a CodeCommit Git repo.
func IsCodeCommitURL(url string) bool {
	u, err := nurl.Parse(url)
	if err != nil {
		return false
	}
	return RegionRe.MatchString(u.Host)
}

//NewCloneURL return CloneURL object for CodeCommit
func NewCloneURL(sess *session.Session, url string) (*CloneURL, error) {
	var err error
	if creds, err := sess.Config.Credentials.Get(); err == nil {
		c := &CloneURL{
			RawURL:     url,
			CredValues: creds,
		}
		if err = c.setURL(); err == nil {
			return c, nil
		}
	}
	return nil, err
}

type CloneURL struct {
	RawURL     string
	CredValues credentials.Value
	u          *nurl.URL
}

func (c *CloneURL) setURL() error {
	u, err := nurl.Parse(c.RawURL)
	if err != nil {
		return err
	}
	c.u = u
	return nil
}

//
func (c *CloneURL) buildCloneURL() error {
	err := c.setURL()
	if err != nil {
		return err
	}
	if c.u.User == nil && strings.Split(c.u.Hostname(), ".")[0] == "git-codecommit" {
		if err = c.addCodeCommitCreds(); err != nil {
			return err
		}
	}
	return err
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

	ctx := NewSigningContext(c.u, region, endpoints.CodecommitServiceID, c.CredValues, time.Now())
	username := c.CredValues.AccessKeyID
	if c.CredValues.SessionToken != "" {
		username = fmt.Sprintf("%s%%%s", username, c.CredValues.SessionToken)
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

type CodeCommitCredentials struct {
	Username string
	Password string
}
