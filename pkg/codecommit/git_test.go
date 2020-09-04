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

	contentExpected := []byte("foo\n")
	baseFile, err := createFile(repoRoot, contentExpected)
	if err != nil {
		t.Fatalf("Failed to create a temp file, err=%v", err)
	}
	gitAdd(t, *baseFile)
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

	filename := filepath.Join(destDir, *baseFile)
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

	contentExpected := []byte("foo\n")
	baseFile, err := createFile(repoRoot, contentExpected)
	if err != nil {
		t.Fatalf("Failed to create a temp file, err=%v", err)
	}

	gitAdd(t, *baseFile)

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

	contentExpected := []byte("foo\n")
	baseFile, err := createFile(cloneDir, contentExpected)
	if err != nil {
		t.Fatalf("Failed to create a temp file, err=%v", err)
	}

	gitAdd(t, *baseFile)
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
	assertFileContents(t, filepath.Join(otherDir, *baseFile), contentExpected)
}

func TestRepoWrapper_AddAll(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("%v", err)
	}

	defer os.Chdir(cwd)

	tempdir := tempDir(t, "TestRepoWrapper-")
	defer os.RemoveAll(tempdir)

	repoRoot := filepath.Join(tempdir, "repo")
	gitInit(t, repoRoot)

	unstagedFile, err := createFile(repoRoot, []byte("hey"))
	if err != nil {
		t.Fatalf("failed to create a file: %v", err)
	}

	addAllPath, err := ioutil.TempDir(repoRoot, "")
	if err != nil {
		t.Fatalf("failed to create a repo directory, err=%v", err)
	}

	stagedFile1, err := createFile(addAllPath, []byte("hey"))
	if err != nil {
		t.Fatalf("failed to create a file: %v", err)
	}

	stagedFile2, err := createFile(addAllPath, []byte("I'm a file"))
	if err != nil {
		t.Fatalf("failed to create a file: %v", err)
	}

	wrapper := &RepoWrapper{}
	repo, err := wrapper.repo(repoRoot)
	if err != nil {
		t.Fatalf("Failed to open a repo, err=%v", err)
	}

	repoDir, err := filepath.Rel(repoRoot, addAllPath)
	if err != nil {
		t.Fatalf("Failed to get the relative path, err=%v", err)
	}
	if err := wrapper.AddAll(repo, repoDir); err != nil {
		t.Fatalf("failed to add the files to the repo, err=%v", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get the working tree, err=%v", err)
	}

	status, err := w.Status()
	if err != nil {
		t.Fatalf("Failed to get the working tree sttus, err=%v", err)
	}

	var stagedFiles []string
	var untrackedFiles []string

	for file, stat := range status {
		switch stat.Staging {
		case git.Added:
			stagedFiles = append(stagedFiles, file)
		case git.Untracked:
			untrackedFiles = append(untrackedFiles, file)
		default:
			t.Fatal("there should only be added and untracked files")
		}
	}

	if len(stagedFiles) != 2 {
		t.Fatal("There should have been 2 staged files")
	}

	for _, stagedFile := range stagedFiles {
		match := false
		for _, expectedStagedFile := range []string{*stagedFile1, *stagedFile2} {
			if filepath.Join(repoDir, expectedStagedFile) == stagedFile {
				match = true
				break
			}
		}
		if !match {
			t.Fatalf("Found unexpected staged file: %s", stagedFile)
		}
	}

	if len(untrackedFiles) != 1 {
		t.Fatal("There should have been only 1 untracked file")
	}
	if untrackedFiles[0] != *unstagedFile {
		t.Fatalf("found unexpected untracked file %s", untrackedFiles[0])
	}
}

func createFile(dir string, contentExpected []byte) (*string, error) {
	f, err := ioutil.TempFile(dir, "")
	if err != nil {
		return nil, err
	}
	baseFile := filepath.Base(f.Name())

	if _, err := f.Write(contentExpected); err != nil {
		return nil, err
	}

	return &baseFile, nil
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
