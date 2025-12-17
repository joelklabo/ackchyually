# Agent CLI CI plan (auth-free)

This document describes what we can reliably test in CI **without authenticating** to OpenAI/Anthropic/GitHub.

## Scope

We care about one outcome: when an agent CLI executes `git`/`bd`/`gh` via `PATH`, it should resolve to `ackchyually` shims and preserve PTY correctness.

In CI we can validate:

1. **Our config mutations are correct and reversible** (TOML/JSON/wrapper).
2. **Our PTY behavior is correct** (existing `TestPTY` suite).
3. **Some CLIs can be smoke-tested without auth** (Codex `sandbox`).

We cannot reliably validate, in CI, that “real agent sessions” for Claude Code or Copilot CLI run tools end-to-end without authenticating.

## What runs in CI

### Always (existing quality gates)

- `go test ./...`
- `go test ./... -run TestPTY -count=1`
- `golangci-lint run` (if available)

### Always (new tests)

**Config mutation unit tests (no external CLIs required):**

- TOML: apply/undo `shell_environment_policy` edits against fixtures
- JSON: apply/undo `~/.claude/settings.json` edits against fixtures
- Wrapper: install/undo Copilot wrapper behavior against a fake “copilot” binary in `t.TempDir()`

These tests prove our integration code is idempotent, reversible, and safe under malformed inputs, even when the real CLIs are not installed.

### Optional / best-effort (auth-free smoke)

#### Codex CLI (`codex sandbox`)

Codex CLI provides `codex sandbox`, which runs arbitrary commands inside OS sandboxing (macOS Seatbelt, Linux Landlock/seccomp). This subcommand is *not* an LLM call and should be auth-free. See:

- Codex CLI reference: https://developers.openai.com/codex/cli/reference
- Codex security guide: https://developers.openai.com/codex/security

**Proposed CI smoke test:**

1. Install Codex CLI (e.g. `npm i -g @openai/codex`)
2. Run `ackchyually integrate codex`
3. Run: `codex sandbox -- which git`
4. Assert output contains the shim path (e.g. `~/.local/share/ackchyually/shims/git`)

**Notes / risks:**

- Linux sandboxing depends on kernel support for Landlock/seccomp; in some CI environments this may be unavailable. If `codex sandbox` fails due to platform restrictions, skip with an explicit “unsupported sandbox” reason.
- macOS `sandbox-exec` exists on GitHub-hosted runners.

We should keep this smoke behind feature detection:

- If `codex` is not installed: skip.
- If `codex sandbox` is unsupported: skip.
- If the runner blocks sandboxing: skip (log why).

## What does NOT run in CI

### Claude Code (`claude`)

Claude Code sessions typically require authentication and may be gated by account/org settings. We can test file mutation and `--help`/`--version` only.

### Copilot CLI (`copilot`)

Copilot CLI requires GitHub authentication for interactive sessions and an active subscription; CI runs will not have either.

We still test:

- Wrapper install/undo behavior
- Status detection logic (installed/not installed)
- Staleness detection logic (version changes / PATH drift)

## Local-only verification

For the full end-to-end “agent runs a tool” guarantee for Claude Code and Copilot CLI, we ship local scripts that:

1. Run `ackchyually integrate all`
2. Ask the user to run one minimal agent prompt that triggers `git --version` or `git status`
3. Validate it shows up in `ackchyually best --tool git`

See: `scripts/verify_agents_local.*` (added in a later task).

