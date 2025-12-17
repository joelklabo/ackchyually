# Local agent verification (Claude Code / Copilot CLI)

This is a **local-only** verification flow for confirming that your agent CLI is actually using the ackchyually shims when it runs tools like `git`.

Why local-only:
- Claude Code and Copilot CLI typically require authentication to run.
- CI should not authenticate to LLM providers.

## What this verifies

1. Shims exist for common tools (`git`, `gh`, `bd`, `xcodebuild`)
2. Agent integrations are installed (`ackchyually integrate all`)
3. An agent-run command results in a recorded successful invocation (visible via `ackchyually best --tool git`)

## Run

```sh
./scripts/verify_agents_local.sh
```

The script:
- Installs shims
- Runs `ackchyually integrate all`
- Creates a fresh temporary directory
- Asks you to run your agent in that directory and have it execute `git --version`
- Checks that `ackchyually best --tool git` is no longer empty

## Expected output (example)

You should see something like:

- `ackchyually integrate verify all` prints `codex: ok`, `claude: ok`, `copilot: ok` (or `skipped` if not installed)
- In the temp dir, `ackchyually best --tool git` prints a list that includes `git --version` (or other `git` commands the agent ran)

If it fails:
- Run `ackchyually integrate status --scan-logs` (best-effort, safe summary) and see `docs/agent_cli_troubleshooting.md`.

