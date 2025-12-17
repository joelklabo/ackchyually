# Troubleshooting: agent CLIs not using ackchyually shims

This doc is for the most common failure: you installed shims, but your agent CLI still runs `/usr/bin/git` (or similar) and `ackchyually` doesn’t log/suggest anything.

## Quick diagnosis (copy/paste)

1) Check what ackchyually thinks is integrated:
```sh
ackchyually integrate status
```

2) In the agent session, check what `git` (and `bd`) resolves to:
```sh
which git
which bd
```

Expected (macOS/Linux):
- `~/.local/share/ackchyually/shims/git`
- `~/.local/share/ackchyually/shims/bd`

Windows (PowerShell):
```powershell
(Get-Command git).Source
(Get-Command bd).Source
```

3) Trigger a known command and verify it shows up:
```sh
git --version
ackchyually best --tool git
```

If `which git` does **not** point to the shim, the agent is bypassing your shim path.

## Common failure modes

### 1) The agent doesn’t inherit your `PATH`
Some agent CLIs run tools in a sanitized environment. If `PATH` isn’t forwarded (or is overwritten), `git` won’t resolve to the shim.

Fix:
```sh
ackchyually integrate all
ackchyually integrate status
```

### 2) The agent runs tools by absolute path
If the agent executes `/usr/bin/git` instead of `git`, `PATH` shims cannot intercept it.

What to do:
- Update your agent rules/prompt to run `git` (not `/usr/bin/git`).
- If the agent has a “tool allowlist” or “command policy”, prefer command names over absolute paths.

### 3) Sandbox restrictions / PATH policy limitations
Some sandboxes don’t respect the host `PATH` (or ignore policy-based PATH changes). In that case:
- `ackchyually integrate verify <tool>` can still confirm config/wrapper integration.
- The most reliable signal is what actually ran (see log inspection below).

## Tool-specific checks

### Codex CLI
1) Re-run integration and verify:
```sh
ackchyually integrate codex
ackchyually integrate verify codex
```

2) Inspect Codex logs (high level):
- `~/.codex/history.jsonl`
- `~/.codex/sessions/`

Look for whether tool executions reference:
- the shim path (`~/.local/share/ackchyually/shims/git`) or
- non-shim absolute paths (like `/usr/bin/git`).

If your ackchyually build supports it, run:
```sh
ackchyually integrate status --scan-logs
```

### Claude Code
1) Re-run integration and verify:
```sh
ackchyually integrate claude
ackchyually integrate verify claude
```

2) Claude Code settings are typically in `~/.claude/settings.json`. `ackchyually integrate claude` manages the `env` section so tools run with a shim-first `PATH`.

### Copilot CLI
1) Re-run integration and verify:
```sh
ackchyually integrate copilot
ackchyually integrate verify copilot
```

2) Copilot CLI stores state under `~/.copilot/`. If your ackchyually build supports it, `ackchyually integrate status --scan-logs` will scan for evidence of shim vs non-shim tool paths.

## Reverting integration

All integrations support `--undo`:
```sh
ackchyually integrate codex --undo
ackchyually integrate claude --undo
ackchyually integrate copilot --undo
```

Then re-check:
```sh
ackchyually integrate status
```

