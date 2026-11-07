package scan

import (
	"os"
	"strings"

	"github.com/svenluijten/snoei/internal/git"
)

type Type string

const (
	TypeBranch   Type = "branch"
	TypeWorktree Type = "worktree"
)

type Item struct {
	Branch string `json:"branch"`
	Type   Type   `json:"type"`
	Path   string `json:"path,omitempty"`
	// Dirty is true when the worktree has uncommitted changes, or the branch has
	// commits that were not on the remote branch when it was deleted.
	Dirty bool `json:"dirty"`
	// Missing is true when the worktree directory was deleted without "git worktree remove".
	Missing bool `json:"missing"`
	// Skip holds the reason this item can never be removed; empty when removable.
	Skip string `json:"skip,omitempty"`
}

// Scan returns local branches whose upstream no longer exists on the remote.
func Scan(dir string, fetch bool) ([]Item, error) {
	// Remote tips are only known before pruning, so unpushed commits are detected only when Scan fetches.
	remoteTips := map[string]string{}
	if fetch {
		out, err := git.Run(dir, "for-each-ref", "refs/remotes", "--format=%(refname)%00%(objectname)")
		if err != nil {
			return nil, err
		}
		for _, line := range strings.Split(out, "\n") {
			ref, sha, _ := strings.Cut(line, "\x00")
			remoteTips[ref] = sha
		}

		if _, err := git.Run(dir, "fetch", "--prune", "--quiet"); err != nil {
			return nil, err
		}
	}

	current, err := git.Run(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, err
	}

	worktrees, mainWorktree, err := listWorktrees(dir)
	if err != nil {
		return nil, err
	}

	refs, err := git.Run(dir, "for-each-ref", "refs/heads", "--format=%(refname:short)%00%(upstream:track)%00%(upstream)")
	if err != nil {
		return nil, err
	}

	var items []Item
	for _, line := range strings.Split(refs, "\n") {
		fields := strings.Split(line, "\x00")
		if len(fields) != 3 || fields[1] != "[gone]" {
			continue
		}
		branch, upstream := fields[0], fields[2]

		item, err := newItem(branch, worktrees[branch], mainWorktree, current)
		if err != nil {
			return nil, err
		}

		// The pruned remote tip is still in the object store, so check that the branch is contained in it.
		if tip := remoteTips[upstream]; tip != "" && !item.Dirty {
			_, err := git.Run(dir, "merge-base", "--is-ancestor", branch, tip)
			item.Dirty = err != nil
		}

		items = append(items, item)
	}

	return items, nil
}

func newItem(branch, path, mainWorktree, current string) (Item, error) {
	if path == "" {
		return Item{Branch: branch, Type: TypeBranch}, nil
	}

	if path == mainWorktree {
		return Item{Branch: branch, Type: TypeBranch, Skip: "checked out in main worktree"}, nil
	}

	item := Item{Branch: branch, Type: TypeWorktree, Path: path}
	if path == current {
		item.Skip = "current worktree"
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		item.Missing = true
		return item, nil
	}

	status, err := git.Run(path, "status", "--porcelain")
	if err != nil {
		return Item{}, err
	}
	item.Dirty = status != ""

	return item, nil
}

// listWorktrees maps branch names to worktree paths and returns the main worktree path.
func listWorktrees(dir string) (map[string]string, string, error) {
	out, err := git.Run(dir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, "", err
	}

	byBranch := map[string]string{}
	var mainWorktree, path string
	for _, line := range strings.Split(out, "\n") {
		if p, ok := strings.CutPrefix(line, "worktree "); ok {
			path = p
			if mainWorktree == "" {
				mainWorktree = p
			}
		}
		if ref, ok := strings.CutPrefix(line, "branch refs/heads/"); ok {
			byBranch[ref] = path
		}
	}

	return byBranch, mainWorktree, nil
}
