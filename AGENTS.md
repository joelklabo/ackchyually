# Agent Instructions (ackchyually)

## Non-negotiables
- PTY correctness is a product requirement. Do not merge changes that weaken interactive behavior.
- PTY integration tests must pass on macOS + Linux for every PR.

## Task tracking (Beads)
We use Beads: https://github.com/steveyegge/beads
- Use `bd` for all non-trivial work.
- Pick only ready tasks: `bd ready`
- Add blockers/deps: `bd dep add <blocked> <blocker> --type blocks`
- Close tasks with PR links.
- This repo uses Beads in `no-db` mode: `.beads/issues.jsonl` is the source of truth (and `.beads/deletions.jsonl` when present).
- Optional (recommended): run `scripts/beads_setup_git.sh` once per clone to enable a smarter merge driver for `.beads/*.jsonl`.

## Quality gates
Before marking work "done":
1) go test ./...
2) go test ./... -run TestPTY -count=1
3) golangci-lint run (if installed)
4) Update Beads issue state + notes.

## Repo conventions
- Keep deps minimal.
- Prefer stdlib unless there’s a strong reason.
- Shim path: ~/.local/share/ackchyually/shims
- DB path: ~/.local/share/ackchyually/ackchyually.sqlite
