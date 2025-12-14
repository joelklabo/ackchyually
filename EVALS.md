# Evals

This repo includes small, local evals to judge whether **ackchyually** helps an agent avoid reaching for `--help` / `-h` when a CLI
invocation is wrong.

## Helpcount eval

Runs a handful of real CLI scenarios (currently: `git` + `go`) in two modes:

- **baseline**: no prior "this worked before here" memory is seeded
- **memory**: a known-good command is seeded first, then the "bad" command is run

For each run we record:

- Whether the scenario ended in success
- How many invocations matched `--help` / `-h` / `help` (from the ackchyually SQLite log)

### Run

```sh
just eval-helpcount
```

JSON output:

```sh
go run ./cmd/ackchyually-eval -json
```

Filter scenarios:

```sh
go run ./cmd/ackchyually-eval -scenario git
```

### Add scenarios

Edit `internal/eval/helpcount/scenarios.go` and add a new `Scenario` to `BuiltinScenarios()`.

Guidelines:

- Use real system tools when possible (e.g. `git`, `go`) so the behavior matches real-world agent sessions.
- Keep scenarios non-interactive and deterministic (no prompts).
- In baseline mode, avoid running successful tool invocations through shims during setup (or you'll accidentally seed memory).
