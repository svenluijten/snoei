package picker

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/svenluijten/snoei/internal/scan"
)

func Pick(items []scan.Item) ([]scan.Item, error) {
	var options []huh.Option[scan.Item]
	for _, item := range items {
		options = append(options, huh.NewOption(label(item), item).Selected(!item.Dirty))
	}

	var selected []scan.Item
	err := huh.NewMultiSelect[scan.Item]().
		Title("Remove merged branches").
		Description("space: toggle · enter: confirm · ctrl+c: cancel").
		Options(options...).
		Value(&selected).
		Run()
	if errors.Is(err, huh.ErrUserAborted) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	dirty := 0
	for _, item := range selected {
		if item.Dirty {
			dirty++
		}
	}
	if dirty == 0 {
		return selected, nil
	}

	confirmed := false
	err = huh.NewConfirm().
		Title(fmt.Sprintf("%d selected worktree(s) have uncommitted changes. Delete them anyway?", dirty)).
		Affirmative("Delete").
		Negative("Keep them").
		Value(&confirmed).
		Run()
	if errors.Is(err, huh.ErrUserAborted) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if confirmed {
		return selected, nil
	}

	var clean []scan.Item
	for _, item := range selected {
		if !item.Dirty {
			clean = append(clean, item)
		}
	}

	return clean, nil
}

func label(item scan.Item) string {
	s := fmt.Sprintf("%-8s %s", item.Type, item.Branch)
	if item.Missing {
		s += "  [folder missing]"
	}
	if item.Dirty {
		s += "  [dirty]"
	}
	if item.Path != "" {
		s += "  " + item.Path
	}

	return s
}
