# snoei

Remove local branches and worktrees that were already merged on GitHub.

```
$ snoei --list
TYPE      BRANCH              STATUS          PATH
branch    fix/login-redirect  clean
worktree  feature/invoices    dirty           /Users/me/Code/app.feature-invoices
worktree  feature/search      folder missing  /Users/me/Code/app.feature-search

$ snoei --yes
removed branch fix/login-redirect
skipped feature/invoices: has uncommitted changes or unpushed commits (use --force)
removed worktree feature/search (/Users/me/Code/app.feature-search)
```

Run `snoei` without flags to pick what to remove from a list.

## Installation

With Homebrew (macOS and Linux):

```
brew install svenluijten/tap/snoei
```

Or download a binary from the [releases page](https://github.com/svenluijten/snoei/releases), or use the GitHub CLI:

```
gh release download -R svenluijten/snoei -p 'snoei_darwin_arm64.tar.gz' -O - | tar xz -C ~/.local/bin snoei
```

Replace `darwin_arm64` with `darwin_amd64`, `linux_amd64` or `linux_arm64` to match your machine.

Or install with Go:

```
go install github.com/svenluijten/snoei/cmd/snoei@latest
```

## Usage

```
snoei [path] [flags]
```

`path` is the repository to clean up. It defaults to the current directory.

| Flag         | Effect                                                     |
|--------------|------------------------------------------------------------|
| `--list`     | Print merged branches and exit                             |
| `--json`     | Print as JSON (with `--list`)                              |
| `-y, --yes`  | Remove all clean merged branches without the picker        |
| `--dry-run`  | Show what would be removed                                 |
| `--force`    | Also remove items with uncommitted changes or unpushed commits (with `--yes`) |
| `--no-fetch` | Skip `git fetch --prune`                                   |

Without a terminal (in scripts or CI), pass `--list` or `--yes`.

## How it works

snoei runs `git fetch --prune`, then finds local branches whose remote branch no longer exists. GitHub deletes the remote branch after a pull request is merged, so these are usually the merged ones. A branch that was never pushed is never listed.

Removing a worktree runs `git worktree remove`, then `git branch -D`.

## Safety

- Worktrees with uncommitted changes, and branches with commits that were never pushed before the remote branch was deleted, are kept unless you pass `--force`, or confirm a second prompt in the picker. Unpushed commits are only detected when snoei runs the fetch itself, so not with `--no-fetch` or after another `git fetch --prune`.
- The branch checked out in the main worktree, and the worktree you run snoei from, are never removed.
- A branch whose pull request was closed without merging also has its remote branch deleted, so it is listed too. Check the list or use `--dry-run` first.

## Exit codes

| Code | Meaning                 |
|------|-------------------------|
| 0    | Success                 |
| 1    | A git command failed    |
| 2    | Invalid flags or usage  |

## License

MIT. See [LICENSE](LICENSE).
