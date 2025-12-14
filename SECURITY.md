# Security Policy

## Supported Versions

We support the latest released version from GitHub Releases.

## Reporting a Vulnerability

Please **do not** open a public issue for security reports.

Use GitHub Security Advisories:
[https://github.com/joelklabo/ackchyually/security/advisories/new](https://github.com/joelklabo/ackchyually/security/advisories/new)

## What this project logs

ackchyually records command invocations (argv + context) and tails of output for debugging/suggestions.

We apply redaction before writing to the local database and stricter redaction on export, but:

- Treat your local DB as sensitive.
- Prefer environment variables/files for secrets rather than literal tokens on the command line.
