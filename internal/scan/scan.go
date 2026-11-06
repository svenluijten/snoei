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
	Dirty  bool   `json:"dirty"`
	// Missing is true when the worktree directory was deleted without "git worktree remove".
	Missing bool `json:"missing"`
	// Skip holds the reason this item can never be removed; empty when removable.
	Skip string `json:"skip,omitempty"`
}

// Scan returns local branches whose upstream no longer exists on the remote.
func Scan(dir string, fetch bool) ([]Item, error) {
	if fetch {
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

	refs, err := git.Run(dir, "for-each-ref", "refs/heads", "--format=%(refname:short)%00%(upstream:track)")
	if err != nil {
		return nil, err
	}

	var items []Item
	for _, line := range strings.Split(refs, "\n") {
		branch, track, _ := strings.Cut(line, "\x00")
		if track != "[gone]" {
			continue
		}

		item, err := newItem(branch, worktrees[branch], mainWorktree, current)
		if err != nil {
			return nil, err
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
