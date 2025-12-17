# Supported versions policy (Codex CLI / Claude Code / Copilot CLI)

This document defines what versions we claim to support, how we test them, and how we keep the list current.

## Guiding principles

- **Prefer configuration stability over chasing “latest”**: we support versions that keep the relevant config file formats stable.
- **Avoid auth in CI**: CI only tests auth-free checks and config mutation correctness.
- **Be explicit**: if we cannot run an authenticated end-to-end test in CI, we document that and provide local verification steps.
- **Minimize breakage**: integrations must be reversible and not clobber user config.

## What “supported” means

For a given CLI + version, we claim:

1. `ackchyually integrate <tool>` can apply and undo integration safely.
2. `ackchyually integrate status` correctly reports installed/integrated/stale.
3. `ackchyually integrate verify` can run **at least one** auth-free sanity check where possible.
4. For CLIs that require auth, we provide a **local-only** end-to-end verification script.

## Version ranges (policy)

We track versions in a machine-readable manifest (see `docs/agent_cli_versions.json` in a later task), but the policy is:

### Codex CLI

- Package: `@openai/codex`
- Supported range: **latest stable + the previous 2 minor releases** (rolling window), *as long as* `shell_environment_policy` is still supported.
- CI: runs config mutation tests always; runs `codex sandbox` smoke when feasible.

### Claude Code

- Package: `@anthropic-ai/claude-code`
- Supported range: **latest stable + previous 2 minor releases**, as long as `~/.claude/settings.json` `env` behavior remains supported.
- CI: config mutation tests only (no authenticated agent runs).
- Local-only: end-to-end verification scripts.

### Copilot CLI

- Package: `@github/copilot`
- Supported range: **latest stable + previous 2 minor releases**.
- CI: wrapper install/undo unit tests (no authenticated agent runs).
- Local-only: end-to-end verification scripts.

## CI test matrix strategy

We do not attempt a combinatorial matrix across many versions.

Instead:

- Pin **one** version per CLI for “installable + `--help` works” smoke checks.
- Pin **one** Codex version for `codex sandbox` smoke (best effort).
- Run config mutation tests against **fixtures** that cover multiple config shapes (this gives broader confidence than installing many versions).

If a breaking change occurs in any tool, we update:

- The fixture corpus
- The integration logic
- The version manifest

## Update checklist (monthly or on breakage)

1. Bump pinned versions (CI) in `.github/workflows/ci.yml` (or the dedicated workflow).
2. Run `go test ./...` and `go test ./... -run TestPTY -count=1`.
3. Run local E2E scripts (requires auth for Claude/Copilot):
   - `scripts/verify_agents_local.sh`
4. Update `docs/agent_cli_versions.json` (manifest) with:
   - new pinned versions
   - observed compatibility notes
5. Update README “Agent CLI” section to match the manifest.

