#!/bin/sh
set -eu

if ! command -v bd >/dev/null 2>&1; then
  echo "bd (Beads) not found; install: https://github.com/steveyegge/beads" >&2
  exit 1
fi

ROOT="$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd -P)"
cd "$ROOT"

if [ ! -d ".git" ]; then
  echo "expected to run inside a git repo: $ROOT" >&2
  exit 1
fi

if [ ! -f ".beads/issues.jsonl" ]; then
  echo "Initializing Beads (no-db) in $ROOT"
  bd init --quiet --no-db --prefix ackchyually
fi

ensure_epic() {
  title="$1"
  desc="$2"

  n="$(bd count --type epic --title "$title" | tr -d '[:space:]' || true)"
  if [ "$n" != "0" ]; then
    echo "OK: $title"
    return 0
  fi

  echo "Create: $title"
  bd create --type epic --priority P0 --title "$title" --description "$desc" >/dev/null
}

ensure_epic "Epic: Bootstrap repo + docs + policies" "Bootstrap repo with docs, policies, and community files."
ensure_epic "Epic: Shim management + resolver" "Shim install/uninstall/doctor and recursion-proof resolver."
ensure_epic "Epic: Exec engine (pipes + PTY)" "PTY-first execution for TTY, pipes for non-TTY."
ensure_epic "Epic: SQLite store + schema" "Local SQLite storage and schema migrations."
ensure_epic "Epic: Tool identity + version cache" "Track exe path, sha256 fingerprint, and cached version."
ensure_epic "Epic: Redaction + safe export" "Redact before DB write; stricter redaction on export."
ensure_epic "Epic: Query/tag/export commands" "Implement best/tag/export user-facing commands."
ensure_epic "Epic: Suggestions on usage-ish failures" "Conservative suggestion output on usage-like errors."
ensure_epic "Epic: CI + security workflows" "CI/lint/security checks for macOS + Linux."
ensure_epic "Epic: goreleaser releases" "Release pipeline and artifacts with checksums."
ensure_epic "Epic: Website + installer + Pages" "ackchyually.sh site, install script, and Pages deploy."

echo "Done."
