#!/bin/sh
set -eu

usage() {
  cat >&2 <<'EOF'
Sync Beads issues to GitHub Issues by attaching `external_ref=gh-<number>`.

Default is dry-run (prints what would happen).

Usage:
  scripts/beads_github_issues.sh [--repo OWNER/REPO] [--apply] [--include-closed]

Options:
  --repo OWNER/REPO     Target repo (defaults to $GITHUB_REPOSITORY or current gh repo)
  --apply               Create GitHub issues + write external_ref back into Beads
  --include-closed      Also create issues for Beads issues with status=closed
EOF
  exit 2
}

REPO="${GITHUB_REPOSITORY:-}"
APPLY=0
INCLUDE_CLOSED=0

while [ $# -gt 0 ]; do
  case "$1" in
    --repo)
      [ $# -ge 2 ] || usage
      REPO="$2"
      shift 2
      ;;
    --apply)
      APPLY=1
      shift
      ;;
    --include-closed)
      INCLUDE_CLOSED=1
      shift
      ;;
    -h|--help)
      usage
      ;;
    *)
      echo "unknown option: $1" >&2
      usage
      ;;
  esac
done

command -v bd >/dev/null 2>&1 || { echo "missing bd (Beads) in PATH" >&2; exit 1; }
command -v gh >/dev/null 2>&1 || { echo "missing gh (GitHub CLI) in PATH" >&2; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo "missing python3 in PATH" >&2; exit 1; }

if [ -z "$REPO" ]; then
  REPO="$(gh repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null || true)"
fi
if [ -z "$REPO" ]; then
  echo "could not determine repo; pass --repo OWNER/REPO or set GITHUB_REPOSITORY" >&2
  exit 1
fi

export REPO APPLY INCLUDE_CLOSED

python3 - <<'PY'
import json
import os
import re
import subprocess
import sys

repo = os.environ["REPO"]
apply = os.environ.get("APPLY", "0") == "1"
include_closed = os.environ.get("INCLUDE_CLOSED", "0") == "1"

issues_path = os.path.join(".beads", "issues.jsonl")
try:
  f = open(issues_path, "r", encoding="utf-8")
except FileNotFoundError:
  print(f"missing {issues_path} (run bd init / scripts/beads_bootstrap.sh first)", file=sys.stderr)
  sys.exit(1)

gh_ref_re = re.compile(r"^gh-(\d+)$")
gh_url_re = re.compile(r"/issues/(\d+)(?:\s*)$")

def run(cmd, *, stdin_text=None):
  return subprocess.run(
    cmd,
    input=stdin_text,
    text=True,
    stdout=subprocess.PIPE,
    stderr=subprocess.PIPE,
    check=False,
  )

def mk_body(beads_issue):
  lines = []
  lines.append("<!-- beads -->")
  lines.append(f"- Beads ID: `{beads_issue.get('id','')}`")
  lines.append(f"- Type: `{beads_issue.get('issue_type','')}`")
  lines.append(f"- Priority: `{beads_issue.get('priority','')}`")
  lines.append(f"- Status: `{beads_issue.get('status','')}`")
  lines.append("")
  desc = beads_issue.get("description") or ""
  if desc.strip():
    lines.append(desc.rstrip())
    lines.append("")
  lines.append("_This issue was created from Beads. The canonical dependency graph lives in `.beads/issues.jsonl`._")
  return "\n".join(lines)

created = 0
skipped = 0
for line in f:
  line = line.strip()
  if not line:
    continue
  try:
    issue = json.loads(line)
  except json.JSONDecodeError:
    continue

  status = issue.get("status", "")
  if status == "closed" and not include_closed:
    continue

  ext = issue.get("external_ref", "") or ""
  if gh_ref_re.match(ext):
    skipped += 1
    continue

  title = issue.get("title") or ""
  if not title.strip():
    continue

  beads_id = issue.get("id") or ""
  body = mk_body(issue)

  if not apply:
    print(f"DRY RUN: would create GitHub issue for {beads_id}: {title!r} (repo={repo})")
    continue

  p = run(
    ["gh", "issue", "create", "--repo", repo, "--title", title, "--body-file", "-"],
    stdin_text=body,
  )
  if p.returncode != 0:
    print(f"gh issue create failed for {beads_id}:", file=sys.stderr)
    print(p.stderr.strip(), file=sys.stderr)
    sys.exit(p.returncode)

  url = p.stdout.strip()
  m = gh_url_re.search(url)
  if not m:
    print(f"could not parse issue number from gh output: {url!r}", file=sys.stderr)
    sys.exit(1)

  num = m.group(1)
  ext_ref = f"gh-{num}"

  u = run(["bd", "update", beads_id, "--external-ref", ext_ref])
  if u.returncode != 0:
    print(f"bd update failed for {beads_id}:", file=sys.stderr)
    print(u.stderr.strip(), file=sys.stderr)
    sys.exit(u.returncode)

  created += 1
  print(f"Created {ext_ref} for {beads_id}: {url}")

if not apply:
  print("DRY RUN: pass --apply to create issues and write external_ref back into Beads.")
else:
  print(f"Done. created={created} skipped_existing={skipped}")
PY

