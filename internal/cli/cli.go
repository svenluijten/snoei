package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/svenluijten/snoei/internal/remove"
	"github.com/svenluijten/snoei/internal/scan"
)

// Picker returns the items the user confirmed for removal, including any dirty ones.
type Picker func(items []scan.Item) ([]scan.Item, error)

type Env struct {
	Stdout      io.Writer
	Stderr      io.Writer
	Interactive bool
	Pick        Picker
}

type options struct {
	dir     string
	list    bool
	json    bool
	yes     bool
	dryRun  bool
	force   bool
	noFetch bool
}

const usage = `Usage: snoei [path] [flags]

List and remove local branches and worktrees whose remote branch is gone.

Flags:
  --list       print merged branches and exit
  --json       print as JSON (with --list)
  -y, --yes    remove all clean merged branches without the picker
  --dry-run    show what would be removed
  --force      also remove items with uncommitted changes (with --yes)
  --no-fetch   skip "git fetch --prune"
`

func Run(args []string, env Env) int {
	opts, err := parse(args, env.Stderr)
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	if err != nil {
		fmt.Fprintln(env.Stderr, "snoei:", err)
		return 2
	}

	if opts.json && !opts.list {
		fmt.Fprintln(env.Stderr, "snoei: --json works only with --list")
		return 2
	}

	if !opts.list && !opts.yes && !env.Interactive {
		fmt.Fprintln(env.Stderr, "snoei: not a terminal; pass --list or --yes")
		return 2
	}

	items, err := scan.Scan(opts.dir, !opts.noFetch)
	if err != nil {
		fmt.Fprintln(env.Stderr, "snoei:", err)
		return 1
	}

	if opts.list {
		return list(items, opts.json, env)
	}

	if opts.yes {
		return removeAll(items, opts, env)
	}

	return pickAndRemove(items, opts, env)
}

func parse(args []string, stderr io.Writer) (options, error) {
	var opts options
	fs := flag.NewFlagSet("snoei", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, usage) }
	fs.BoolVar(&opts.list, "list", false, "")
	fs.BoolVar(&opts.json, "json", false, "")
	fs.BoolVar(&opts.yes, "yes", false, "")
	fs.BoolVar(&opts.yes, "y", false, "")
	fs.BoolVar(&opts.dryRun, "dry-run", false, "")
	fs.BoolVar(&opts.force, "force", false, "")
	fs.BoolVar(&opts.noFetch, "no-fetch", false, "")

	// The standard flag package stops at the first positional argument, so parse again after it.
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return opts, err
		}
		if fs.NArg() == 0 {
			break
		}
		positional = append(positional, fs.Arg(0))
		args = fs.Args()[1:]
	}

	if len(positional) > 1 {
		return opts, fmt.Errorf("expected at most one path, got %d", len(positional))
	}

	opts.dir = "."
	if len(positional) == 1 {
		opts.dir = positional[0]
	}

	return opts, nil
}

func list(items []scan.Item, asJSON bool, env Env) int {
	if asJSON {
		if items == nil {
			items = []scan.Item{}
		}
		enc := json.NewEncoder(env.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(items); err != nil {
			fmt.Fprintln(env.Stderr, "snoei:", err)
			return 1
		}
		return 0
	}

	if len(items) == 0 {
		fmt.Fprintln(env.Stdout, "No merged branches found.")
		return 0
	}

	w := tabwriter.NewWriter(env.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TYPE\tBRANCH\tSTATUS\tPATH")
	for _, item := range items {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", item.Type, item.Branch, status(item), item.Path)
	}
	w.Flush()

	return 0
}

func status(item scan.Item) string {
	if item.Skip != "" {
		return "skip: " + item.Skip
	}
	if item.Missing {
		return "folder missing"
	}
	if item.Dirty {
		return "dirty"
	}

	return "clean"
}

func removeAll(items []scan.Item, opts options, env Env) int {
	if len(items) == 0 {
		fmt.Fprintln(env.Stdout, "No merged branches found.")
		return 0
	}

	code := 0
	for _, item := range items {
		if item.Skip != "" {
			fmt.Fprintf(env.Stdout, "skipped %s: %s\n", item.Branch, item.Skip)
			continue
		}
		if item.Dirty && !opts.force {
			fmt.Fprintf(env.Stdout, "skipped %s: has uncommitted changes (use --force)\n", item.Branch)
			continue
		}
		if !removeOne(item, opts.dir, opts.force, opts.dryRun, env) {
			code = 1
		}
	}

	return code
}

func pickAndRemove(items []scan.Item, opts options, env Env) int {
	var removable []scan.Item
	for _, item := range items {
		if item.Skip != "" {
			fmt.Fprintf(env.Stdout, "skipped %s: %s\n", item.Branch, item.Skip)
			continue
		}
		removable = append(removable, item)
	}

	if len(removable) == 0 {
		fmt.Fprintln(env.Stdout, "No merged branches found.")
		return 0
	}

	selected, err := env.Pick(removable)
	if err != nil {
		fmt.Fprintln(env.Stderr, "snoei:", err)
		return 1
	}

	code := 0
	for _, item := range selected {
		// The picker already asked for confirmation on dirty items.
		if !removeOne(item, opts.dir, true, opts.dryRun, env) {
			code = 1
		}
	}

	return code
}

func removeOne(item scan.Item, dir string, force, dryRun bool, env Env) bool {
	label := fmt.Sprintf("%s %s", item.Type, item.Branch)
	if item.Path != "" {
		label += " (" + item.Path + ")"
	}

	if dryRun {
		fmt.Fprintln(env.Stdout, "would remove", label)
		return true
	}

	if err := remove.Remove(dir, item, force); err != nil {
		fmt.Fprintln(env.Stderr, "snoei:", err)
		return false
	}

	fmt.Fprintln(env.Stdout, "removed", label)

	return true
}
