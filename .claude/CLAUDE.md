# snoei

Go CLI that lists and removes local branches and worktrees whose remote branch is gone (merged on GitHub).

## Commands

- Test: `go test ./...`
- Format: `gofmt -w .`
- Build: `go build -o snoei ./cmd/snoei`

## Conventions

- "Merged" means the upstream shows as `[gone]` after `git fetch --prune`. Do not add GitHub API calls for this.
- Call git through `internal/git.Run`. Do not use libgit2 or go-git.
- Tests run against real temporary repos built with `internal/testrepo` (a clone plus a bare `origin`). Do not mock git.
- The interactive picker is passed into `cli.Run` as a `cli.Picker` function, so CLI tests replace it with a plain function. Keep TTY code in `internal/picker`, which has no tests.
- Exit codes: 0 success, 1 runtime or git error, 2 usage error.
- An item with `Skip` set is never removed, not even with `--force`.

## Releasing

Push a `v*` tag (for example `git tag v0.1.0 && git push origin v0.1.0`). `.github/workflows/release.yml` runs the tests, then GoReleaser publishes darwin and linux binaries (amd64 and arm64) to a GitHub release.
