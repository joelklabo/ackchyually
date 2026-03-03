#!/usr/bin/env bash
set -euo pipefail

echo "Agent CLI smoke checks (auth-free)"

echo
echo "Versions:"
codex --version
claude --version
copilot --version

tmp_home="$(mktemp -d)"
trap 'rm -rf "$tmp_home"' EXIT
export HOME="$tmp_home"

echo
echo "Using HOME=$HOME"

echo
echo "Integrate status (before):"
go run ./cmd/ackchyually integrate status

go run ./cmd/ackchyually integrate codex
go run ./cmd/ackchyually integrate claude
go run ./cmd/ackchyually integrate copilot

echo
echo "Integrate status (after):"
go run ./cmd/ackchyually integrate status

# Install git shim after integration to avoid temp dir issues
go run ./cmd/ackchyually shim install git

echo
echo "Verify:"
go run ./cmd/ackchyually integrate verify all
