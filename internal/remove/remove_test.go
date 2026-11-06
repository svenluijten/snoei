package remove_test

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/svenluijten/snoei/internal/remove"
	"github.com/svenluijten/snoei/internal/scan"
	"github.com/svenluijten/snoei/internal/testrepo"
)

func TestRemoveDeletesBranch(t *testing.T) {
	r := testrepo.New(t)
	r.Branch("feature/merged")
	r.Merge("feature/merged")

	err := remove.Remove(r.Dir, scan.Item{Branch: "feature/merged", Type: scan.TypeBranch}, false)
	if err != nil {
		t.Fatal(err)
	}

	if out := r.Git(r.Dir, "branch", "--list", "feature/merged"); out != "" {
		t.Errorf("branch still exists: %q", out)
	}
}

func TestRemoveDeletesWorktreeAndItsBranch(t *testing.T) {
	r := testrepo.New(t)
	wt := r.Worktree("feature/merged")
	r.Merge("feature/merged")

	err := remove.Remove(r.Dir, scan.Item{Branch: "feature/merged", Type: scan.TypeWorktree, Path: wt}, false)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(wt); !os.IsNotExist(err) {
		t.Errorf("worktree directory still exists: %v", err)
	}
	if out := r.Git(r.Dir, "branch", "--list", "feature/merged"); out != "" {
		t.Errorf("branch still exists: %q", out)
	}
}

func TestRemoveRefusesDirtyWorktreeWithoutForce(t *testing.T) {
	r := testrepo.New(t)
	wt := r.Worktree("feature/dirty")
	r.Merge("feature/dirty")
	r.Dirty(wt)

	err := remove.Remove(r.Dir, scan.Item{Branch: "feature/dirty", Type: scan.TypeWorktree, Path: wt, Dirty: true}, false)
	if !errors.Is(err, remove.ErrDirty) {
		t.Fatalf("got %v, want ErrDirty", err)
	}

	if _, err := os.Stat(wt); err != nil {
		t.Errorf("worktree was removed: %v", err)
	}
}

func TestRemoveDeletesDirtyWorktreeWithForce(t *testing.T) {
	r := testrepo.New(t)
	wt := r.Worktree("feature/dirty")
	r.Merge("feature/dirty")
	r.Dirty(wt)

	err := remove.Remove(r.Dir, scan.Item{Branch: "feature/dirty", Type: scan.TypeWorktree, Path: wt, Dirty: true}, true)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(wt); !os.IsNotExist(err) {
		t.Errorf("worktree directory still exists: %v", err)
	}
}

func TestRemoveRefusesSkippedItemEvenWithForce(t *testing.T) {
	r := testrepo.New(t)

	err := remove.Remove(r.Dir, scan.Item{Branch: "main", Type: scan.TypeBranch, Skip: "checked out in main worktree"}, true)
	if !errors.Is(err, remove.ErrSkipped) {
		t.Fatalf("got %v, want ErrSkipped", err)
	}
}

func TestRemoveDeletesWorktreeWithDeletedDirectoryAndItsBranch(t *testing.T) {
	r := testrepo.New(t)
	wt := r.Worktree("feature/gone-dir")
	r.Merge("feature/gone-dir")
	if err := os.RemoveAll(wt); err != nil {
		t.Fatal(err)
	}

	err := remove.Remove(r.Dir, scan.Item{Branch: "feature/gone-dir", Type: scan.TypeWorktree, Path: wt, Missing: true}, false)
	if err != nil {
		t.Fatal(err)
	}

	if out := r.Git(r.Dir, "worktree", "list"); strings.Contains(out, wt) {
		t.Errorf("worktree still registered:\n%s", out)
	}
	if out := r.Git(r.Dir, "branch", "--list", "feature/gone-dir"); out != "" {
		t.Errorf("branch still exists: %q", out)
	}
}
