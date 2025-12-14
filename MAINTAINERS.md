# Maintainer checklist (first-time setup)

This is a one-time checklist for setting up the GitHub repo, GitHub Pages, and basic protections.

## Repo + Pages

1) Create the repo: `joelklabo/ackchyually`
2) Enable GitHub Pages:
   - Settings → Pages
   - Build and deployment → Source: **GitHub Actions**
3) Add DNS for `ackchyually.sh` (at your DNS provider) and set the custom domain in GitHub Pages:
   - Ensure `site/CNAME` is present and correct
   - Configure DNS records per GitHub Pages’ custom domain instructions
4) Add branch protections (Settings → Branches) for `main`:
   - Require status checks to pass (CI + Lint)
   - Require pull request reviews
5) (Optional) Enable Discussions
6) (Optional) Configure Codecov if needed:
   - If Codecov requires a token, add `CODECOV_TOKEN` as a GitHub Actions secret
   - (Optional) With `gh`:
     - `printf '%s' \"$CODECOV_TOKEN\" | gh secret set CODECOV_TOKEN -R joelklabo/ackchyually`

## Optional automation (gh)

If you have the GitHub CLI authenticated with a token that can edit the repo, you can automate parts of the setup.

Note: if `GITHUB_TOKEN` is set in your environment, it can override `gh`’s stored auth; run commands as `GITHUB_TOKEN= gh ...`
to force using your normal `gh auth login` token.

```sh
# Enable discussions
GITHUB_TOKEN= gh repo edit joelklabo/ackchyually --enable-discussions

# Enable GitHub Pages (build_type=workflow means “GitHub Actions”)
GITHUB_TOKEN= gh api -X POST repos/joelklabo/ackchyually/pages -f build_type=workflow

# Set custom domain (HTTPS enforcement may fail until DNS is live + cert is issued)
GITHUB_TOKEN= gh api -X PUT repos/joelklabo/ackchyually/pages -f cname=ackchyually.sh

# Protect main (require CI + Lint + 1 review)
GITHUB_TOKEN= gh api -X PUT repos/joelklabo/ackchyually/branches/main/protection --input - <<'JSON'
{
  "required_status_checks": {
    "strict": true,
    "contexts": [
      "test (ubuntu-latest)",
      "test (macos-latest)",
      "golangci"
    ]
  },
  "enforce_admins": false,
  "required_pull_request_reviews": {
    "required_approving_review_count": 1
  },
  "restrictions": null
}
JSON
```
