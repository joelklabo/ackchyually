# Contributing to ackchyually

Thanks for helping.

## Project principles
- PTY correctness is a **hard product requirement**.
- Keep dependencies minimal.
- Never block the wrapped command on logging/DB failures.
- Be conservative with suggestions/auto-exec (default: off).

## Dev prerequisites
- Go 1.22+
- (Optional) `just`
- (Optional) `golangci-lint`
- (Optional) Beads task tool: https://github.com/steveyegge/beads

## Quickstart
```sh
git clone https://github.com/joelklabo/ackchyually
cd ackchyually
go mod download
go test ./...
```

Using `just`:
```sh
just test
just test-pty
just lint
```

## PTY tests (non-negotiable)
Run locally:
```sh
go test ./... -run TestPTY -count=1
```

If PTY tests are flaky, fix the flake before adding features.

## Working on tasks (Beads)
- Pick ready work: `bd ready`
- Create a task: `bd create "Task: ..." -p 1`
- Add blockers: `bd dep add <blocked> <blocker> --type blocks`
- Close with PR link.

Beads data lives in `.beads/issues.jsonl` (committed). After cloning, run:
```sh
scripts/beads_setup_git.sh
```

Optional: mirror Beads issues into GitHub Issues (writes `external_ref=gh-<number>` back into Beads):
```sh
scripts/beads_github_issues.sh --repo joelklabo/ackchyually
scripts/beads_github_issues.sh --repo joelklabo/ackchyually --apply
```

## Running locally with shims
```sh
go install ./cmd/ackchyually
ackchyually shim install git
export PATH="$HOME/.local/share/ackchyually/shims:$PATH"
git status
```

## Release process (maintainers)
```sh
git tag v0.1.0 && git push origin v0.1.0
```

Local release dry run:
```sh
goreleaser release --snapshot --clean
```

## Website
Site lives in `site/`, deployed via GitHub Pages.
Install script source: `scripts/install.sh`
Deployed copy: `site/install.sh`

The Pages workflow syncs `scripts/install.sh` → `site/install.sh` before deploy.

Manual sync (for local preview):
```sh
just site-sync-install
```
