package testrepo

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Repo is a local clone with a bare "origin" next to it.
type Repo struct {
	t      *testing.T
	Dir    string
	Remote string
}

func New(t *testing.T) *Repo {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")

	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	r := &Repo{t: t, Dir: filepath.Join(root, "repo"), Remote: filepath.Join(root, "origin.git")}
	r.Git(root, "init", "--bare", "-b", "main", r.Remote)
	r.Git(root, "clone", r.Remote, r.Dir)
	r.Git(r.Dir, "checkout", "-b", "main")
	r.Commit(r.Dir, "initial")
	r.Git(r.Dir, "push", "-u", "origin", "main")

	return r
}

func (r *Repo) Git(dir string, args ...string) string {
	r.t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}

	return strings.TrimSpace(string(out))
}

func (r *Repo) Commit(dir, msg string) {
	r.t.Helper()
	r.Git(dir, "commit", "--allow-empty", "-m", msg)
}

// Branch creates a pushed branch with one commit, without checking it out.
func (r *Repo) Branch(name string) {
	r.t.Helper()
	r.Git(r.Dir, "branch", name, "main")
	r.Git(r.Dir, "push", "-u", "origin", name)
}

// Worktree creates a pushed branch checked out in a new worktree and returns its path.
func (r *Repo) Worktree(name string) string {
	r.t.Helper()
	path := filepath.Join(filepath.Dir(r.Dir), "wt-"+strings.ReplaceAll(name, "/", "-"))
	r.Git(r.Dir, "worktree", "add", "-b", name, path, "main")
	r.Commit(path, "work on "+name)
	r.Git(path, "push", "-u", "origin", name)

	return path
}

// Merge simulates a merged pull request: the remote branch is deleted and pruned locally.
func (r *Repo) Merge(name string) {
	r.t.Helper()
	r.Git(r.Dir, "push", "origin", "--delete", name)
}

func (r *Repo) Dirty(dir string) {
	r.t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("x"), 0o644); err != nil {
		r.t.Fatal(err)
	}
}
