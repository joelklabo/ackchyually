# Beads - AI-Native Issue Tracking

Welcome to Beads! This repository uses **Beads** for issue tracking - a modern, AI-native tool designed to live directly in your codebase alongside your code.

## What is Beads?

Beads is issue tracking that lives in your repo, making it perfect for AI coding agents and developers who want their issues close to their code. No web UI required - everything works through the CLI and integrates seamlessly with git.

**Learn more:** [github.com/steveyegge/beads](https://github.com/steveyegge/beads)

## Quick Start

This repository uses Beads in **no-db mode**: `.beads/issues.jsonl` is the source of truth (no SQLite required).

### Essential Commands

```bash
# Create new issues
bd create "Add user authentication"

# View all issues
bd list

# View issue details
bd show <issue-id>

# Update issue status
bd update <issue-id> --status in_progress
bd update <issue-id> --status closed

# Sync with git remote
bd sync
```

### Repo notes / workarounds

This repo is configured for `no-db` mode, but some `bd sync` flows may still require SQLite:

```bash
# Flush JSONL in no-db mode (workaround for "not in a bd workspace"):
bd sync --flush-only --db .beads/beads.db

# One-way sync from main on ephemeral branches:
BD_NO_DB=0 bd sync --from-main --db .beads/beads.db --no-pull --no-push
```

`bd blocked` may not show issues with `status=blocked`; use:

```bash
bd list --status blocked
```

`bd ready` may include issues that still have `blocks` dependencies; treat dependencies in `bd show <issue-id>` as authoritative.

### Recommended git merge driver
For nicer merges of `.beads/issues.jsonl`, run once per clone:

```bash
scripts/beads_setup_git.sh
```

## GitHub Issues (optional)

Beads issues can be linked to GitHub Issues using the `external_ref` field.

This repo includes a helper to create GitHub Issues for Beads issues and write back `external_ref=gh-<number>`:

```bash
scripts/beads_github_issues.sh --repo joelklabo/ackchyually
scripts/beads_github_issues.sh --repo joelklabo/ackchyually --apply
```

### Working with Issues

Issues in Beads are:
- **Git-native**: Stored in `.beads/issues.jsonl` and synced like code
- **AI-friendly**: CLI-first design works perfectly with AI coding agents
- **Branch-aware**: Issues can follow your branch workflow
- **Always in sync**: Auto-syncs with your commits

## Why Beads?

✨ **AI-Native Design**
- Built specifically for AI-assisted development workflows
- CLI-first interface works seamlessly with AI coding agents
- No context switching to web UIs

🚀 **Developer Focused**
- Issues live in your repo, right next to your code
- Works offline, syncs when you push
- Fast, lightweight, and stays out of your way

🔧 **Git Integration**
- Automatic sync with git commits
- Branch-aware issue tracking
- Intelligent JSONL merge resolution

## Get Started with Beads

Try Beads in your own projects:

```bash
# Install Beads
curl -sSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash

# Initialize in your repo
bd init

# Create your first issue
bd create "Try out Beads"
```

## Learn More

- **Documentation**: [github.com/steveyegge/beads/docs](https://github.com/steveyegge/beads/tree/main/docs)
- **Quick Start Guide**: Run `bd quickstart`
- **Examples**: [github.com/steveyegge/beads/examples](https://github.com/steveyegge/beads/tree/main/examples)

---

*Beads: Issue tracking that moves at the speed of thought* ⚡
