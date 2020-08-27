package codecommit

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// TestRepoWrapperCloneEmpty tests RepoWrapper.Clone() of an empty Git repo.
func TestRepoWrapperCloneEmpty(t *testing.T) {
	tempdir := tempDir(t, "TestRepoWrapper-")
	defer os.RemoveAll(tempdir)
	repoRoot := filepath.Join(tempdir, "repo")
	gitInit(t, repoRoot)

	repoWrapper := RepoWrapper{}
	destDir := filepath.Join(repoRoot, "dest")
	r, isEmpty, err := repoWrapper.Clone(repoRoot, destDir)
	if err != nil {
		t.Fatalf("Failed cloning repo %v, err=%v, repo=%v", repoRoot, err, r)

	}
	if !isEmpty {
		t.Fatalf("Repo %v should have been empty", repoRoot)
	}

}

// TestRepoWrapperClone tests RepoWrapper.Clone() a non-empty Git repo, verifying that the contents of the clone matches the source repo.
func TestRepoWrapperClone(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("%v", err)
	}

	defer os.Chdir(cwd)

	tempdir := tempDir(t, "TestRepoWrapper-")
	defer os.RemoveAll(tempdir)
	repoRoot := filepath.Join(tempdir, "repo")
	gitInit(t, repoRoot)

	os.Chdir(repoRoot)

	f, err := ioutil.TempFile(repoRoot, "")
	if err != nil {
		t.Fatalf("%v", err)
	}

	baseFile := filepath.Base(f.Name())
	contentExpected := []byte("foo\n")
	f.Write(contentExpected)
	gitAdd(t, baseFile)
	gitCommit(t, "commit it")

	repoWrapper := RepoWrapper{}
	destDir := filepath.Join(repoRoot, "dest")
	r, isEmpty, err := repoWrapper.Clone(repoRoot, destDir)
	if err != nil {
		t.Fatalf("Failed cloning repo %v, err=%v, repo=%v", repoRoot, err, r)

	}
	if isEmpty {
		t.Fatalf("Repo %v should not have been empty", repoRoot)
	}

	filename := filepath.Join(destDir, baseFile)
	_, err = os.Stat(filename)
	if err != nil {
		t.Fatalf("Expected file %v was not cloned, err=%v", filename, err)
	}

	assertFileContents(t, filename, contentExpected)
}

// TestRepoWrapperCommit tests RepoWrapper.commit() of a single file, verifying that the commitl Log() method returns the expected commit.
func TestRepoWrapperCommit(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("%v", err)
	}

	defer os.Chdir(cwd)

	tempdir := tempDir(t, "TestRepoWrapper-")
	defer os.RemoveAll(tempdir)
	repoRoot := filepath.Join(tempdir, "repo")
	gitInit(t, repoRoot)

	os.Chdir(repoRoot)

	f, err := ioutil.TempFile(repoRoot, "")
	if err != nil {
		t.Fatalf("%v", err)
	}

	baseFile := filepath.Base(f.Name())
	contentExpected := []byte("foo\n")
	f.Write(contentExpected)
	gitAdd(t, baseFile)

	repoWrapper := RepoWrapper{}
	repo, err := repoWrapper.repo(repoRoot)
	if err != nil {
		t.Fatalf("Failed to get a new repo, err=%v", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get a WorkTree for repo %v, err=%v", repo, err)
	}

	name := "TestIt"
	email := "testit@foo.local"
	message := "testit"
	h, err := repoWrapper.Commit(w, name, email, message, false)
	if err != nil {
		t.Fatalf("Failed to commit to repo %v, err=%v", repo, err)
	}

	cIter, err := repo.Log(&git.LogOptions{From: *h})
	err = cIter.ForEach(func(c *object.Commit) error {
		if c.Author.Name != name {
			t.Fatalf("Expected name %v not found for %v", name, h)
		}
		if c.Author.Email != email {
			t.Fatalf("Expected email %v not found for %v", email, h)
		}
		if c.Message != message {
			t.Fatalf("Expected message %v not found for %v", message, h)
		}
		return nil
	})
}

// TestRepoWrapperPush tests that the RepoWrapper can push changes to a bare remote Git repo.
func TestRepoWrapperPush(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("%v", err)
	}

	defer os.Chdir(cwd)

	tempdir := tempDir(t, "TestRepoWrapper-")
	defer os.RemoveAll(tempdir)

	repoRoot := filepath.Join(tempdir, "repo")
	gitInit(t, repoRoot, "--bare")

	cloneDir := filepath.Join(tempdir, "clone")
	gitClone(t, repoRoot, cloneDir)

	os.Chdir(cloneDir)

	f, err := ioutil.TempFile(cloneDir, "")
	if err != nil {
		t.Fatalf("%v", err)
	}
	baseFile := filepath.Base(f.Name())
	contentExpected := []byte("foo\n")
	f.Write(contentExpected)
	gitAdd(t, baseFile)
	gitCommit(t, "commit it")

	repoWrapper := RepoWrapper{}
	repo, err := repoWrapper.repo(cloneDir)
	if err != nil {
		t.Fatalf("Failed to get a new repo, err=%v", err)
	}
	err = repoWrapper.PushR(repo)
	if err != nil {
		t.Fatalf("Failed to push repo %v to %v, err=%v", cloneDir, repoRoot, err)
	}

	otherDir := filepath.Join(tempdir, "other")
	gitClone(t, repoRoot, otherDir)
	assertFileContents(t, filepath.Join(otherDir, baseFile), contentExpected)
}

func assertFileContents(t *testing.T, filename string, expected []byte) {
	actual, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("Error reading %v, err=%v", expected, err)
	}

	if bytes.Compare(actual, expected) != 0 {
		t.Fatalf("Contents of %v does not match expected %v, actual=%v", filename, expected, actual)
	}

}
func tempDir(t *testing.T, prefix string) string {
	t.Helper()
	tempdir, err := ioutil.TempDir("", prefix)
	if err != nil {
		t.Fatalf("Temp directory creation failed, err=%v", err)
	}
	return tempdir
}

func gitAdd(t *testing.T, files ...string) {
	t.Helper()
	execGit(t, "add", files...)
}

func gitInit(t *testing.T, args ...string) {
	t.Helper()
	execGit(t, "init", args...)
}

func gitClone(t *testing.T, url string, dest string) {
	t.Helper()
	execGit(t, "clone", url, dest)
}

func gitCommit(t *testing.T, message string) {
	t.Helper()
	execGit(t, "commit", "-m", message)
}

func execGit(t *testing.T, command string, args ...string) {
	t.Helper()
	a := []string{command}
	a = append(a, args...)
	cmd := exec.Command("git", a...)
	stdouterr, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed, args=%v, err=%v, stdouterr=%s", command, a, err, stdouterr)
	}
}
