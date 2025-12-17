# Agent CLI integration design (Codex CLI / Claude Code / Copilot CLI)

## Problem

`ackchyually` works by putting shim executables (e.g. `git`, `bd`, `gh`) in a shim directory and ensuring that directory is *first* on `PATH`, so tool invocations resolve to the shims.

Agent CLIs often run tools via a subprocess layer that can:

- Filter which environment variables are forwarded (dropping or rewriting `PATH`)
- Run inside a sandbox with a different `PATH`
- Execute absolute tool paths (bypassing `PATH`)

Result: the agent runs `git`/`bd`/`gh`, but it may bypass `ackchyually`, so no commands get recorded.

## Goal

Provide a first-class integration command that makes tool execution inside these agent CLIs resolve to `ackchyually` shims without the user hand-editing rc files:

- Codex CLI (`codex`)
- Claude Code (`claude`)
- GitHub Copilot CLI (`copilot`)

## Non-goals

- IDE integrations beyond these CLIs
- Running authenticated LLM sessions in CI
- Modifying / replacing `git` itself (we integrate via `PATH`)

## Key paths / invariants

- Shim directory: `execx.ShimDir()` (currently `~/.local/share/ackchyually/shims`)
- Integration state: `~/.local/share/ackchyually/integrations/*.json` (new)
- Integrations must be:
  - Idempotent (re-running is safe)
  - Reversible (`--undo`)
  - Safe by default (don’t leak secrets; don’t clobber user config silently)

## Proposed CLI surface

`ackchyually integrate` becomes the single entry point:

- `ackchyually integrate all [--dry-run] [--undo]`
- `ackchyually integrate codex [--dry-run] [--undo]`
- `ackchyually integrate claude [--dry-run] [--undo]`
- `ackchyually integrate copilot [--dry-run] [--undo]`
- `ackchyually integrate status [--json] [--scan-logs]`
- `ackchyually integrate verify [--json]` (auth-free checks where possible)

Conventions:

- `--dry-run` prints the planned filesystem changes without writing.
- `--undo` removes only ackchyually-managed edits; if the target file has changed unexpectedly since integration, the command refuses unless `--force` is provided.
- `status` explains “installed / version / integrated / stale” per tool.

## Shared implementation pieces

### PATH computation (shim-first)

Compute a deterministic shim-first path string:

- `shimFirstPath := pathutil.PrependToPATH(execx.ShimDir(), os.Getenv("PATH"))`
- De-duplicate the shim dir if already present.
- Use `filepath.ListSeparator` (cross-platform).

This path string is persisted into tool config during integration and stored in integration state to support staleness warnings.

### Integration state file

Each tool has a state JSON file (example `integrations/codex.json`):

```json
{
  "tool": "codex",
  "integrated_at": "2025-12-17T12:34:56Z",
  "tool_version": "x.y.z",
  "shim_dir": "/Users/me/.local/share/ackchyually/shims",
  "shim_first_path": "/Users/me/.local/share/ackchyually/shims:/opt/homebrew/bin:...",
  "config_path": "/Users/me/.codex/config.toml",
  "config_sha256_before": "…",
  "config_sha256_after": "…"
}
```

This is used by `status` and for `--undo` safety checks.

## Tool-specific integration mechanisms

### Codex CLI (`codex`)

**What we change**

Codex reads `~/.codex/config.toml` and supports `shell_environment_policy`, including setting explicit environment values for all spawned subprocesses (see: https://developers.openai.com/codex/local-configuration).

We add or modify:

- `[shell_environment_policy]`
  - `set.PATH = "<shimFirstPath>"`
  - If `include_only` exists, ensure it contains `PATH` and `HOME` (do not introduce `include_only` if not present to avoid breaking existing flows).

**Why this works**

Even if Codex filters environment variables, `shell_environment_policy.set` injects the desired `PATH` into every subprocess it launches.

**Idempotence**

- Re-running sets `set.PATH` to the same computed value if unchanged.
- We track `config_sha256_after` in integration state; if the file has changed later, we still merge safely, but `--undo` is conservative.

**Undo**

Undo removes only ackchyually-owned changes:

- If current config hash equals the recorded `config_sha256_after`, restore `config_sha256_before` by re-applying the inverse mutation (preferred), or restoring from an on-disk backup.
- If config has changed since integration, refuse unless `--force`.

**Example resulting config**

```toml
[shell_environment_policy]
set = { PATH = "/Users/me/.local/share/ackchyually/shims:/opt/homebrew/bin:/usr/bin:/bin" }
```

### Claude Code (`claude`)

**What we change**

Claude Code supports `~/.claude/settings.json` with an `env` object applied to every session (see: https://docs.anthropic.com/en/docs/claude-code/settings).

We add or modify:

- `env.PATH = "<shimFirstPath>"`

We do not modify unrelated permissions or sandbox settings.

**Undo**

- If we created `env.PATH`, delete it.
- If we overwrote an existing `env.PATH`, restore it from integration state (or refuse unless `--force` if the file has drifted).

**Example resulting settings**

```json
{
  "env": {
    "PATH": "/Users/me/.local/share/ackchyually/shims:/opt/homebrew/bin:..."
  }
}
```

### GitHub Copilot CLI (`copilot`)

**Constraints**

Copilot CLI has a `config.json` file in `~/.copilot` (or under `XDG_CONFIG_HOME`) but documentation does not describe a supported “inject PATH into tool subprocesses” setting (see: https://docs.github.com/en/copilot/how-tos/use-copilot-in-the-cli).

**What we do instead**

Install a reversible wrapper *for the `copilot` executable itself* so that when the user launches Copilot CLI, it runs with a shim-first `PATH`:

- Create `copilot.real` as the original target binary
- Install a wrapper script named `copilot` that:
  - Computes `shimFirstPath` from the *current* environment
  - Exports `PATH=shimFirstPath`
  - `exec`s `copilot.real "$@"`

We only do this if `copilot` is a normal file we can move in place (or a symlink we can replace safely). If we can’t do it safely, we fall back to printing a one-time instruction (or requiring `ackchyually shim enable`).

**Undo**

- Remove wrapper and restore original binary/symlink.

## Verification strategy

### Auth-free verification (CI-safe)

- **Config mutation tests**: unit tests that validate we can apply and undo changes on fixture files for TOML/JSON.
- **Binary presence tests**: `--version` / `--help` checks for installed CLIs.
- **Codex sandbox smoke** (preferred): run `codex sandbox -- which git` and assert it prints the shim path. (Exact invocation may vary by CLI version; keep behind feature detection.)

### Auth-required verification (local-only)

For Claude/Copilot, full “agent runs tool and we observe shim usage” likely requires user authentication. Provide local scripts that:

1. Run `ackchyually integrate all`
2. Ask the user to run one minimal agent prompt that triggers `git --version`
3. Confirm `ackchyually best --tool git` includes a successful invocation

## Staleness & proactive hints

We consider an integration “stale” if any of:

- Tool version differs from the recorded integrated version
- Shim dir differs (user moved HOME)
- For file-based integrations: config file hash differs from recorded `sha256_after`
- `shimFirstPath` differs materially from what we recorded (PATH changed)

When stale or missing:

- `ackchyually integrate status` reports “not integrated” or “stale”.
- If we detect the agent CLIs are installed, `ackchyually` prints a rate-limited hint during normal shim usage:
  - “Detected Codex CLI installed; run `ackchyually integrate codex` to ensure shims are used inside Codex.”
