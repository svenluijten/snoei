package scan_test

import (
	"os"
	"testing"

	"github.com/svenluijten/snoei/internal/scan"
	"github.com/svenluijten/snoei/internal/testrepo"
)

func TestScanFindsMergedBranchesAndWorktrees(t *testing.T) {
	r := testrepo.New(t)
	r.Branch("feature/merged")
	r.Branch("feature/open")
	wt := r.Worktree("feature/merged-wt")
	r.Worktree("feature/open-wt")
	r.Merge("feature/merged")
	r.Merge("feature/merged-wt")

	items, err := scan.Scan(r.Dir, true)
	if err != nil {
		t.Fatal(err)
	}

	want := []scan.Item{
		{Branch: "feature/merged", Type: scan.TypeBranch},
		{Branch: "feature/merged-wt", Type: scan.TypeWorktree, Path: wt},
	}
	assertItems(t, items, want)
}

func TestScanIgnoresUnpushedBranches(t *testing.T) {
	r := testrepo.New(t)
	r.Git(r.Dir, "branch", "local-only")

	items, err := scan.Scan(r.Dir, true)
	if err != nil {
		t.Fatal(err)
	}

	assertItems(t, items, nil)
}

func TestScanMarksDirtyWorktrees(t *testing.T) {
	r := testrepo.New(t)
	wt := r.Worktree("feature/dirty")
	r.Merge("feature/dirty")
	r.Dirty(wt)

	items, err := scan.Scan(r.Dir, true)
	if err != nil {
		t.Fatal(err)
	}

	want := []scan.Item{{Branch: "feature/dirty", Type: scan.TypeWorktree, Path: wt, Dirty: true}}
	assertItems(t, items, want)
}

func TestScanSkipsBranchCheckedOutInMainWorktree(t *testing.T) {
	r := testrepo.New(t)
	r.Branch("feature/current")
	r.Git(r.Dir, "checkout", "feature/current")
	r.Merge("feature/current")

	items, err := scan.Scan(r.Dir, true)
	if err != nil {
		t.Fatal(err)
	}

	want := []scan.Item{{Branch: "feature/current", Type: scan.TypeBranch, Skip: "checked out in main worktree"}}
	assertItems(t, items, want)
}

func TestScanSkipsWorktreeItIsRunFrom(t *testing.T) {
	r := testrepo.New(t)
	wt := r.Worktree("feature/here")
	r.Merge("feature/here")

	items, err := scan.Scan(wt, true)
	if err != nil {
		t.Fatal(err)
	}

	want := []scan.Item{{Branch: "feature/here", Type: scan.TypeWorktree, Path: wt, Skip: "current worktree"}}
	assertItems(t, items, want)
}

func TestScanWithoutFetchDoesNotSeeRemoteDeletion(t *testing.T) {
	r := testrepo.New(t)
	r.Branch("feature/merged")
	r.Merge("feature/merged")
	r.Git(r.Dir, "update-ref", "refs/remotes/origin/feature/merged", "main")

	items, err := scan.Scan(r.Dir, false)
	if err != nil {
		t.Fatal(err)
	}

	assertItems(t, items, nil)
}

func assertItems(t *testing.T, got, want []scan.Item) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d items %+v, want %d %+v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("item %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestScanMarksWorktreeWithDeletedDirectoryAsMissing(t *testing.T) {
	r := testrepo.New(t)
	wt := r.Worktree("feature/gone-dir")
	r.Merge("feature/gone-dir")
	if err := os.RemoveAll(wt); err != nil {
		t.Fatal(err)
	}

	items, err := scan.Scan(r.Dir, true)
	if err != nil {
		t.Fatal(err)
	}

	want := []scan.Item{{Branch: "feature/gone-dir", Type: scan.TypeWorktree, Path: wt, Missing: true}}
	assertItems(t, items, want)
}
