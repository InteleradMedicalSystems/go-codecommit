package tests

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/magiconair/properties/assert"
)

var cwd string

func init() {
	// setup plugin cache to make the tests run a bit faster.
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	if os.Getenv("NO_TF_PLUGIN_CACHE") == "" {
		cachedir := filepath.Join(cwd, ".plugin-cache")
		archdir := filepath.Join(cachedir, "linux_amd64")
		err = os.MkdirAll(archdir, os.FileMode(0755))
		if err != nil {
			panic(err)
		}
		os.Setenv("TF_PLUGIN_CACHE_DIR", cachedir)
	}
}

type TestOptions struct {
	srcdir string
	region string
}

func TestCredentialHelper(t *testing.T) {
	t.Parallel()
	e := NewCredentialHelperE2ETest(t, TestOptions{
		srcdir: "terraform",
		region: "us-east-1",
	})
	e.testCredentialHelperClone()
}

type CredentialHelperTest struct {
	t                    *testing.T
	topts                TestOptions
	uniqueid             string
	project              string
	environment          string
	bucketHandlerVersion string
	tempdir              string
	root                 string
}

func (e *CredentialHelperTest) setUp() {
	e.t.Helper()
	//_, root, _, ok := runtime.Caller(0)
	tempdir := tempDir(e.t, ".", "")
	fmt.Printf("TEMPDIR %s\n", tempdir)
	e.tempdir = tempdir
	symlinkTerraRoot(e.t, e.topts.srcdir, e.tempdir)
}

func (e *CredentialHelperTest) tearDown() {
	os.RemoveAll(e.tempdir)
}

func (e *CredentialHelperTest) testCredentialHelperClone() {
	// ensure that we can use the credential-helper to clone a CodeCommit repo.
	e.t.Helper()
	e.setUp()
	defer e.tearDown()

	tempdir := e.tempdir
	opts := e.getOpts(tempdir)
	defer terraform.Destroy(e.t, opts)

	terraform.InitAndApply(e.t, opts)
	e.assertOutputs(opts)

	url := e.expectedCloneURL()
	dst := path.Join(tempdir, e.repositoryName())
	args := []string{
		"--config",
		fmt.Sprintf("credential.helper=!%s credential-helper $@", e.helperExe()),
		"--config=credential.UseHttpPath=true",
	}
	err := gitClone(url, dst, args...)
	if err != nil {
		e.t.Fatal(err)
	}
}

func (e *CredentialHelperTest) helperExe() string {
	return path.Join(cwd, "../build/codecommit")
	//fmt.Sprintf("codecommit-%s-%s", runtime.GOOS, runtime.GOARCH))
}

func (e *CredentialHelperTest) assertOutputs(opts *terraform.Options) {
	e.t.Helper()
	assertOutput(e.t, opts, "clone_url_http", e.expectedCloneURL())
}

func (e *CredentialHelperTest) repositoryName() string {
	if e.uniqueid == "" {
		e.uniqueid = strings.ToLower(fmt.Sprintf("%s-%s-%s", e.project, e.t.Name(), random.UniqueId()))
	}
	return e.uniqueid
}

func (e *CredentialHelperTest) expectedCloneURL() string {
	return fmt.Sprintf("https://git-codecommit.%s.amazonaws.com/v1/repos/%s",
		e.topts.region,
		e.repositoryName())
}

func (e *CredentialHelperTest) getOpts(tempdir string) *terraform.Options {
	opts := &terraform.Options{
		TerraformDir: tempdir,
		Vars: map[string]interface{}{
			"project":         e.project,
			"environment":     e.environment,
			"repository_name": e.repositoryName(),
		},
	}
	return opts
}

func assertOutput(t *testing.T, opts *terraform.Options, name, expected string) {
	t.Helper()
	actual, err := terraform.OutputE(t, opts, name)
	if err != nil {
		t.Error(err)
		return
	}
	assert.Equal(t, actual, expected)
}

func NewCredentialHelperE2ETest(t *testing.T, topts TestOptions) *CredentialHelperTest {
	return &CredentialHelperTest{
		t:           t,
		topts:       topts,
		project:     "go-codecommit",
		environment: getEnvironment(),
	}
}

func symlinkTerraRoot(t *testing.T, srcdir, destdir string) {
	files, err := ioutil.ReadDir(srcdir)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		ext := filepath.Ext(f.Name())
		if ext == ".tf" || ext == ".zip" {
			err = os.Symlink(filepath.Join("..", srcdir, f.Name()), filepath.Join(destdir, f.Name()))
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

func tempDir(t *testing.T, dir, prefix string) string {
	t.Helper()
	if prefix == "" {
		prefix = t.Name() + "-"
	}
	tempdir, err := ioutil.TempDir(dir, prefix)
	if err != nil {
		t.Fatalf("Temp directory creation failed, err=%v", err)
	}
	return tempdir
}

func gitClone(url string, dest string, extraArgs ...string) error {
	args := []string{}
	args = append(args, extraArgs...)
	args = append(args, url)
	args = append(args, dest)
	return execGit("clone", args...)
}

func execGit(command string, args ...string) error {
	a := []string{command}
	a = append(a, args...)
	cmd := exec.Command("git", a...)
	fmt.Printf("git %v\n", strings.Join(a, " "))
	output, err := cmd.CombinedOutput()
	fmt.Printf("%s\n", output)
	return err
}

func getEnvironment() string {
	name := os.Getenv("DEP_ENVIRONMENT")
	if name == "" {
		return "dev"
	}
	return name
}
