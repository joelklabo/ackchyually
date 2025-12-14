#!/bin/sh
set -eu

if ! command -v git >/dev/null 2>&1; then
  echo "missing git in PATH" >&2
  exit 1
fi
if ! command -v bd >/dev/null 2>&1; then
  echo "missing bd (Beads) in PATH" >&2
  exit 1
fi

git config merge.beads.driver "bd merge %A %O %A %B"
git config merge.beads.name "bd JSONL merge driver"

echo "Configured git merge driver: merge.beads"
echo "Note: .gitattributes in this repo routes .beads/*.jsonl to merge=beads."

