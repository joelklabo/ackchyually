# ackchyually

<img src="assets/ackchyually_hero_1600x500.png" alt="ackchyually" width="800" />

![CI](https://github.com/joelklabo/ackchyually/actions/workflows/ci.yml/badge.svg)
![Lint](https://github.com/joelklabo/ackchyually/actions/workflows/lint.yml/badge.svg)
![CodeQL](https://github.com/joelklabo/ackchyually/actions/workflows/codeql.yml/badge.svg)
![govulncheck](https://github.com/joelklabo/ackchyually/actions/workflows/govulncheck.yml/badge.svg)
![Scorecard](https://github.com/joelklabo/ackchyually/actions/workflows/scorecard.yml/badge.svg)
![Release](https://img.shields.io/github/v/release/joelklabo/ackchyually)
![Go Version](https://img.shields.io/github/go-mod/go-version/joelklabo/ackchyually)
![License](https://img.shields.io/github/license/joelklabo/ackchyually)
![Go Report Card](https://goreportcard.com/badge/github.com/joelklabo/ackchyually)
![PkgGoDev](https://pkg.go.dev/badge/github.com/joelklabo/ackchyually.svg)
![Coverage](https://img.shields.io/endpoint?url=https://ackchyually.sh/coverage.json)
![Downloads](https://img.shields.io/github/downloads/joelklabo/ackchyually/total)
![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20linux%20%7C%20windows-blue)

**ackchyually** — the “ackchyually…” friend for your CLI. Remembers what worked (per repo). Suggests the right command when you get the details wrong.

## PTY-first (hard requirement)
If you're in a TTY, ackchyually runs tools under a real PTY. This is non-negotiable for interactive CLIs and agent shells (Claude Code / Codex CLI / Copilot CLI). On Windows, it uses ConPTY.

## Install

### macOS / Linux (One-liner)
```sh
curl -fsSL https://ackchyually.sh/install.sh | sh
```

### Windows (Manual)
Download the latest `ackchyually_Windows_x86_64.zip` (or arm64) from [GitHub Releases](https://github.com/joelklabo/ackchyually/releases), unzip it, and place `ackchyually.exe` in your PATH.

### Install Shims
Once `ackchyually` is installed and in your PATH:

```sh
ackchyually shim install git gh xcodebuild
```

### Verify
Ensure your PATH is configured correctly so that `which git` (Unix) or `Get-Command git` (Windows) points to the shim:

```sh
# Unix
which git
# output should be: ~/.local/share/ackchyually/shims/git

# Windows (PowerShell)
(Get-Command git).Source
# output should be: ...\ackchyually\shims\git.exe
```

If it doesn't point to the shim, follow the instructions printed by `ackchyually shim install` or run:

```sh
ackchyually shim enable
```

## Quickstart
1) **Install shims** for tools you want to track:
   ```sh
   ackchyually shim install git gh xcodebuild
   ```

2) **Use tools normally**:
   ```sh
   git status
   xcodebuild test -scheme App
   ```

3) **Get suggestions** when you make a mistake:
   ```sh
   $ git log -1 --prety=%s
   error: unknown option `prety=%s'
   ackchyually: suggestion (previous success in this repo):
     git log -1 --pretty=%s
   ```

4) **Query what worked**:
   ```sh
   ackchyually best --tool xcodebuild "test"
   ackchyually export --format md --tool xcodebuild
   ```

## What it looks like (when it’s working)

First, confirm your shell is actually using the shims:

```sh
export PATH="$HOME/.local/share/ackchyually/shims:$PATH"
hash -r

which git
# ~/.local/share/ackchyually/shims/git
```

Then, after you’ve run a successful command at least once in this repo/context, ackchyually can help when you make a “usage-ish” mistake:

```sh
$ git log -1 --pretty=%s
fix: something

$ git log -1 --prety=%s
error: unknown option `prety=%s'
usage: git log [<options>] [<revision-range>] [[--] <path>...]
ackchyually: suggestion (previous success in this repo):
  git log -1 --pretty=%s
```

Optional auto-exec (off by default):

```sh
$ export ACKCHYUALLY_AUTO_EXEC=known_success
$ git log -1 --prety=%s
ackchyually: auto-exec (known_success):
  git log -1 --pretty=%s
fix: something
```

## How it works
- Transparent PATH shims (busybox-style symlinks) so you keep typing `git ...` normally.
- Logs invocations to a local SQLite DB (redacted) keyed by repo/cwd context (`~/.local/share/ackchyually/ackchyually.sqlite`).
- On “usage-ish” failures, prints one known-good command that worked before in the same context.

## Integrate with agents (Codex CLI / Claude Code / Copilot CLI)
If you use an agent CLI that runs tools like `git`/`gh`/`bd` via your `PATH`, integrate it so the agent hits the ackchyually shims automatically (no shell rc edits).

```sh
ackchyually shim install git gh bd xcodebuild
ackchyually integrate all
ackchyually integrate status
ackchyually integrate verify all
```

30-second verification checklist (inside the agent session):
```sh
which git
git --version
ackchyually best --tool git
```

Notes:
- This only works when the agent executes tools by name (e.g. `git`), not by absolute path (e.g. `/usr/bin/git`).
- Docs:
  - Codex CLI configuration (`shell_environment_policy`, config file): https://developers.openai.com/codex/configuration
  - Claude Code settings (`~/.claude/settings.json`): https://docs.anthropic.com/en/docs/claude-code/settings
  - Copilot CLI: https://docs.github.com/en/copilot/how-tos/use-copilot-in-the-cli
- Troubleshooting: `docs/agent_cli_troubleshooting.md`

Supported versions (from `internal/integrations/agentclis/supported_versions.json`):
- Codex CLI (`codex`, npm `@openai/codex`): `>= 0.0.0`
- Claude Code (`claude`, npm `@anthropic-ai/claude-code`): `>= 0.0.0`
- Copilot CLI (`copilot`, npm `@github/copilot`): `>= 0.0.0`

Automated check (POSIX): `just test-agent` (or `go test ./... -run TestAgentCLI -count=1`).

## Commands
- `ackchyually shim install <tool...>`
- `ackchyually shim list`
- `ackchyually shim enable`
- `ackchyually shim uninstall <tool...>`
- `ackchyually shim doctor`
- `ackchyually integrate all|status|verify [codex|claude|copilot|all]`
- `ackchyually integrate codex|claude|copilot [--dry-run] [--undo]`
- `ackchyually best --tool <tool> "<query>"`
- `ackchyually tag add "<tag>" -- <command...>`
- `ackchyually tag run "<tag>"`
- `ackchyually export --format md|json [--tool <tool>]`

## Security
- Redaction runs before writing to the local DB.
- Export is stricter (normalizes paths, redacts more).
- Auto-exec is off by default.

### Optional auto-exec (off by default)
If you want ackchyually to automatically re-run the top known-success command on “usage-ish” failures (interactive TTY only):

```sh
export ACKCHYUALLY_AUTO_EXEC=known_success
```

## Development

```sh
just test
just test-pty
just lint
```

Maintainers: see `MAINTAINERS.md`.

## License

MIT
