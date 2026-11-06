package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/svenluijten/snoei/internal/cli"
	"github.com/svenluijten/snoei/internal/scan"
	"github.com/svenluijten/snoei/internal/testrepo"
)

func run(t *testing.T, env cli.Env, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	env.Stdout = &stdout
	env.Stderr = &stderr

	code := cli.Run(args, env)

	return code, stdout.String(), stderr.String()
}

func TestListPrintsTable(t *testing.T) {
	r := testrepo.New(t)
	r.Branch("feature/a")
	wt := r.Worktree("feature/b")
	r.Merge("feature/a")
	r.Merge("feature/b")
	r.Dirty(wt)

	code, out, errOut := run(t, cli.Env{}, r.Dir, "--list")

	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	for _, want := range []string{"feature/a", "branch", "clean", "feature/b", "worktree", "dirty", wt} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestListAcceptsFlagsBeforePath(t *testing.T) {
	r := testrepo.New(t)
	r.Branch("feature/a")
	r.Merge("feature/a")

	code, out, errOut := run(t, cli.Env{}, "--list", r.Dir)

	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	if !strings.Contains(out, "feature/a") {
		t.Errorf("output missing branch:\n%s", out)
	}
}

func TestListPrintsJSON(t *testing.T) {
	r := testrepo.New(t)
	wt := r.Worktree("feature/b")
	r.Merge("feature/b")

	code, out, errOut := run(t, cli.Env{}, r.Dir, "--list", "--json")

	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	var items []scan.Item
	if err := json.Unmarshal([]byte(out), &items); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	want := scan.Item{Branch: "feature/b", Type: scan.TypeWorktree, Path: wt}
	if len(items) != 1 || items[0] != want {
		t.Errorf("got %+v, want [%+v]", items, want)
	}
}

func TestListPrintsEmptyJSONArray(t *testing.T) {
	r := testrepo.New(t)

	_, out, _ := run(t, cli.Env{}, r.Dir, "--list", "--json")

	if strings.TrimSpace(out) != "[]" {
		t.Errorf("got %q, want []", out)
	}
}

func TestYesRemovesCleanAndSkipsDirty(t *testing.T) {
	r := testrepo.New(t)
	r.Branch("feature/clean")
	dirty := r.Worktree("feature/dirty")
	r.Merge("feature/clean")
	r.Merge("feature/dirty")
	r.Dirty(dirty)

	code, out, errOut := run(t, cli.Env{}, r.Dir, "--yes")

	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	if b := r.Git(r.Dir, "branch", "--list", "feature/clean"); b != "" {
		t.Errorf("clean branch still exists")
	}
	if _, err := os.Stat(dirty); err != nil {
		t.Errorf("dirty worktree was removed")
	}
	if !strings.Contains(out, "removed branch feature/clean") || !strings.Contains(out, "skipped feature/dirty") {
		t.Errorf("unexpected output:\n%s", out)
	}
}

func TestYesWithForceRemovesDirty(t *testing.T) {
	r := testrepo.New(t)
	dirty := r.Worktree("feature/dirty")
	r.Merge("feature/dirty")
	r.Dirty(dirty)

	code, _, errOut := run(t, cli.Env{}, r.Dir, "--yes", "--force")

	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	if _, err := os.Stat(dirty); !os.IsNotExist(err) {
		t.Errorf("dirty worktree still exists")
	}
}

func TestDryRunRemovesNothing(t *testing.T) {
	r := testrepo.New(t)
	r.Branch("feature/clean")
	r.Merge("feature/clean")

	code, out, errOut := run(t, cli.Env{}, r.Dir, "--yes", "--dry-run")

	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	if b := r.Git(r.Dir, "branch", "--list", "feature/clean"); b == "" {
		t.Errorf("branch was removed during dry run")
	}
	if !strings.Contains(out, "would remove branch feature/clean") {
		t.Errorf("unexpected output:\n%s", out)
	}
}

func TestNonInteractiveWithoutModeFails(t *testing.T) {
	r := testrepo.New(t)

	code, _, errOut := run(t, cli.Env{Interactive: false}, r.Dir)

	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
	if !strings.Contains(errOut, "--list or --yes") {
		t.Errorf("unexpected error: %s", errOut)
	}
}

func TestJSONWithoutListFails(t *testing.T) {
	r := testrepo.New(t)

	code, _, _ := run(t, cli.Env{}, r.Dir, "--yes", "--json")

	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
}

func TestPickerRemovesOnlySelectedAndExcludesSkipped(t *testing.T) {
	r := testrepo.New(t)
	r.Branch("feature/pick")
	r.Branch("feature/keep")
	r.Branch("feature/current")
	r.Git(r.Dir, "checkout", "feature/current")
	r.Merge("feature/pick")
	r.Merge("feature/keep")
	r.Merge("feature/current")

	var offered []scan.Item
	pick := func(items []scan.Item) ([]scan.Item, error) {
		offered = items
		for _, item := range items {
			if item.Branch == "feature/pick" {
				return []scan.Item{item}, nil
			}
		}
		return nil, nil
	}

	code, _, errOut := run(t, cli.Env{Interactive: true, Pick: pick}, r.Dir)

	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	if len(offered) != 2 {
		t.Errorf("picker offered %+v, want feature/keep and feature/pick only", offered)
	}
	if b := r.Git(r.Dir, "branch", "--list", "feature/pick"); b != "" {
		t.Errorf("picked branch still exists")
	}
	if b := r.Git(r.Dir, "branch", "--list", "feature/keep"); b == "" {
		t.Errorf("unpicked branch was removed")
	}
}

func TestPickerRemovesConfirmedDirtyWorktree(t *testing.T) {
	r := testrepo.New(t)
	dirty := r.Worktree("feature/dirty")
	r.Merge("feature/dirty")
	r.Dirty(dirty)

	pick := func(items []scan.Item) ([]scan.Item, error) { return items, nil }

	code, _, errOut := run(t, cli.Env{Interactive: true, Pick: pick}, r.Dir)

	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	if _, err := os.Stat(dirty); !os.IsNotExist(err) {
		t.Errorf("dirty worktree still exists")
	}
}
