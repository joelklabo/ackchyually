# Gemini Code Assistant Context

This document provides context for the Gemini Code Assistant to understand the `ackchyually` project.

## Project Overview

`ackchyually` is a command-line tool written in Go that acts as a "smart" command history. It remembers successful commands on a per-repository basis and suggests them when a user makes a mistake. It uses a system of shims to intercept commands and log them to a local SQLite database.

The project is structured as a standard Go application, with the main entrypoint in `cmd/ackchyually/main.go`. The core application logic is located in the `internal/app` package. The project uses the `just` command runner for task automation.

## Building and Running

The project uses a `justfile` to define common tasks.

*   **Build:** To build the `ackchyually` binary, run:
    ```bash
    just build
    ```

*   **Test:** To run all tests, use:
    ```bash
    just test
    ```
    To run the PTY-specific tests, use:
    ```bash
    just test-pty
    ```

*   **Lint:** To run the linter, use:
    ```bash
    just lint
    ```

*   **Install Locally:** To install the `ackchyually` binary to your local `GOPATH`, run:
    ```bash
    just install-local
    ```

## Development Conventions

*   **Formatting:** The project uses `gofmt` for code formatting. The `just fmt` command can be used to format the entire codebase.
*   **Linting:** The project uses `golangci-lint` for linting. The `just lint` command can be used to run the linter.
*   **Dependencies:** The project uses Go modules to manage dependencies. The `go.mod` file lists the project's dependencies.
*   **Database:** The application uses a SQLite database to store command history. The database schema is likely defined in the `internal/store` package.
