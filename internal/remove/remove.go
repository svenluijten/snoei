package remove

import (
	"errors"
	"fmt"

	"github.com/svenluijten/snoei/internal/git"
	"github.com/svenluijten/snoei/internal/scan"
)

var (
	ErrDirty   = errors.New("has uncommitted changes or unpushed commits")
	ErrSkipped = errors.New("cannot be removed")
)

func Remove(dir string, item scan.Item, force bool) error {
	if item.Skip != "" {
		return fmt.Errorf("%s: %w: %s", item.Branch, ErrSkipped, item.Skip)
	}

	if item.Dirty && !force {
		return fmt.Errorf("%s: %w (use --force)", item.Branch, ErrDirty)
	}

	if item.Type == scan.TypeWorktree {
		args := []string{"worktree", "remove", item.Path}
		if force || item.Missing {
			args = append(args, "--force")
		}
		if _, err := git.Run(dir, args...); err != nil {
			return err
		}
	}

	// -D because squash-merged branches are not ancestors of the default branch.
	_, err := git.Run(dir, "branch", "-D", item.Branch)

	return err
}
