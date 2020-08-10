package codecommit

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
)

//RepoWrapper wraps basic go-git comands
type RepoWrapper struct {
}

//Clone a Git repo, return true if the repo is up to date or was from an empty clone.
func (r *RepoWrapper) Clone(cloneURL *CloneURL, destDir string) (*git.Repository, bool, error) {
	log.Debugf("Cloning Git repo %s, dest %s", cloneURL, destDir)

	cloneOpts := &git.CloneOptions{
		URL: cloneURL.String(),
	}

	repo, err := git.PlainClone(destDir, false, cloneOpts)
	if err != nil {
		switch err {
		case transport.ErrEmptyRemoteRepository:
			return repo, true, nil
		case git.NoErrAlreadyUpToDate:
			return repo, true, nil
		default:
			return nil, false, err
		}
	}
	return repo, false, nil
}

//Pull a Git repo from path
func (r *RepoWrapper) Pull(path string) error {
	repo, err := r.repo(path)
	if err != nil {
		return err
	}
	return r.PullR(repo)
}

//PullR a Git repo.
func (r *RepoWrapper) PullR(repo *git.Repository) error {
	w, err := repo.Worktree()
	if err != nil {
		return err
	}
	err = w.Pull(&git.PullOptions{RemoteName: "origin"})
	switch err {
	case transport.ErrEmptyRemoteRepository:
		log.Warnf("Warning: %s", err)
		return nil
	case git.NoErrAlreadyUpToDate:
		return nil
	default:
		return err
	}
}

//Push a Git repo from path
func (r *RepoWrapper) Push(path string) error {
	repo, err := r.repo(path)
	if err != nil {
		return err
	}
	return r.PushR(repo)
}

//PushR a Git repo.
func (r *RepoWrapper) PushR(repo *git.Repository) error {
	err := repo.Push(&git.PushOptions{})
	switch err {
	case git.NoErrAlreadyUpToDate:
		log.Warnf("Warning: %s", err)
		return nil
	default:
		return err
	}
}

//Commit to a Git repo
func (r *RepoWrapper) Commit(w *git.Worktree, name, email, message string, force bool) (*plumbing.Hash, error) {
	status, err := w.Status()
	if err != nil {
		return nil, err
	}

	if !status.IsClean() || force {
		log.Info(status.String())
		commit, err := w.Commit(message, &git.CommitOptions{
			Author: &object.Signature{
				Name:  name,
				Email: email,
				When:  time.Now(),
			},
		})
		if err != nil {
			return nil, err
		}
		log.Infof("Committed %s", commit)
		return &commit, nil
	}
	log.Infof("no changes to commit")
	return nil, nil
}

//GetDestPath returns
//get the dest from either last element of args or
//the basename of the url (with the .git suffix stripped).
func (r *RepoWrapper) GetDestPath(path string) string {
	return strings.Replace(filepath.Base(path), ".git", "", -1)
}

func (r *RepoWrapper) manifestMap(destPrefix string, repo *git.Repository) (map[string]*object.File, error) {
	m := make(map[string]*object.File)
	ref, err := repo.Head()
	if err != nil {
		if err != plumbing.ErrReferenceNotFound {
			return m, err
		}
	} else {
		c, err := repo.CommitObject(ref.Hash())
		if err != nil {
			return m, err
		}
		fIter, err := c.Files()
		if err != nil {
			return m, err
		}
		err = fIter.ForEach(func(f *object.File) error {
			log.Debugf("repoFile:%s", f.Name)
			if strings.HasPrefix(f.Name, destPrefix) {
				m[f.Name] = f
			}
			return nil
		})
		if err != nil {
			return m, err
		}
	}
	return m, nil
}

func (r *RepoWrapper) addRemove(destPrefix string, boundary string, repo *git.Repository, fm map[string]os.FileInfo) (*git.Worktree, error) {
	w, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	mm, err := r.manifestMap(destPrefix, repo)
	if err != nil {
		return nil, err
	}
	log.Debugf("%v", mm)

	for fn := range mm {
		target := fn[len(destPrefix)+1:]
		_, ok := fm[target]
		log.Debugf("Target %v, fn %v, destPrefix %v, boundary %v", target, fn, destPrefix, boundary)
		if !ok && strings.HasPrefix(target, boundary) {
			log.Infof("Removing %v", fn)
			w.Remove(fn)
		}
	}

	for fn, f := range fm {
		if f.Mode().IsRegular() {
			target := filepath.Join(destPrefix, fn)
			_, ok := mm[target]
			if !ok {
				// Only log file addition for files that are not in the Git cache.
				log.Infof("Adding %v", target)
			}
			// Always call Add()
			w.Add(target)
		}
	}
	return w, nil
}

func (r *RepoWrapper) repo(path string) (*git.Repository, error) {
	return git.PlainOpen(path)
}
