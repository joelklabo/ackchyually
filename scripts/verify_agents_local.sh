#!/usr/bin/env bash
set -euo pipefail

echo "ackchyually local agent verification (Claude Code / Copilot CLI)"
echo "NOTE: This is LOCAL-ONLY and may require authentication for your agent CLI."
echo

require_cmd() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "error: missing required command: $cmd" >&2
    exit 1
  fi
}

require_cmd ackchyually

if command -v git >/dev/null 2>&1; then
  : # ok
else
  echo "warning: git not found in PATH; you can still verify shim wiring, but git commands won't run." >&2
fi

echo "Step 1/4: Install shims (safe, non-destructive)"
echo "  ackchyually shim install git gh bd xcodebuild"
read -r -p "Press Enter to run this command... " _ || true
ackchyually shim install git gh bd xcodebuild
echo

echo "Step 2/4: Integrate agent CLIs (writes config / wrapper files)"
echo "  ackchyually integrate all"
echo "  ackchyually integrate status"
read -r -p "Press Enter to run integration... " _ || true
ackchyually integrate all
ackchyually integrate status
echo

echo "Step 3/4: Verify integration (auth-free sanity checks)"
echo "  ackchyually integrate verify all"
ackchyually integrate verify all
echo

echo "Step 4/4: Prove the agent is hitting the shims"
echo
echo "We will use a fresh temp directory so 'ackchyually best --tool git' starts empty."
echo "In that directory, start your agent CLI and ask it to run: git --version"
echo

tmp="${TMPDIR:-/tmp}/ackchyually-agent-verify-$$"
mkdir -p "$tmp"
cd "$tmp"

echo "Temp dir: $tmp"
echo
echo "Baseline (should usually say 'no successful commands recorded yet'):"
ackchyually best --tool git || true
echo

cat <<'EOF'
Now, in THIS directory, run ONE of these (requires auth):

- Claude Code:
    claude
  Then ask it:
    Run `git --version`.

- Copilot CLI:
    copilot
  Then ask it:
    Run `git --version`.

If your agent doesn't execute commands directly, run the suggested command yourself.

When done, come back here and press Enter.
EOF

read -r -p "Press Enter after the agent has run 'git --version'... " _ || true
echo

echo "Expected: 'ackchyually best --tool git' should now show a successful git command for this directory."
out="$(ackchyually best --tool git 2>&1 || true)"
echo "$out"
echo

if echo "$out" | grep -qi "no successful commands recorded yet"; then
  echo "error: still no recorded successful git commands in this directory." >&2
  echo "If the agent ran '/usr/bin/git' (absolute path) or did not inherit PATH, ackchyually can't intercept it." >&2
  echo "Try: ackchyually integrate status --scan-logs" >&2
  exit 1
fi

echo "ok: ackchyually recorded a successful git command in the agent verification directory."

