#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
prefix="${ACKCHYUALLY_CI_NPM_PREFIX:-"$repo_root/.ci/npm"}"

mkdir -p "$prefix"

echo "Installing agent CLIs into: $prefix"
echo "node: $(node --version)"
echo "npm:  $(npm --version)"

npm install --prefix "$prefix" --no-fund --no-audit \
  @openai/codex \
  @anthropic-ai/claude-code \
  @github/copilot

bin_dir="$prefix/node_modules/.bin"
if [[ ! -d "$bin_dir" ]]; then
  echo "error: expected npm bin dir missing: $bin_dir" >&2
  exit 1
fi

echo "Adding to PATH: $bin_dir"
export PATH="$bin_dir:$PATH"
if [[ -n "${GITHUB_PATH:-}" ]]; then
  echo "$bin_dir" >>"$GITHUB_PATH"
fi

echo
echo "Installed versions:"
codex --version
claude --version
copilot --version
