#!/bin/sh
set -eu

REPO="joelklabo/ackchyually"
BIN="ackchyually"
BASE_URL="https://github.com/${REPO}/releases"

say() { printf '%s\n' "$*"; }

supports_color() {
  [ -t 1 ] && [ "${NO_COLOR:-}" = "" ] && [ "${TERM:-}" != "dumb" ]
}

# Optional ANSI styling for nicer UX (falls back to plain text).
BOLD=""
DIM=""
GREEN=""
YELLOW=""
RESET=""
if supports_color; then
  BOLD="$(printf '\033[1m')"
  DIM="$(printf '\033[2m')"
  GREEN="$(printf '\033[32m')"
  YELLOW="$(printf '\033[33m')"
  RESET="$(printf '\033[0m')"
fi

usage() {
  echo "usage: install.sh [-b bindir] [version]" >&2
  exit 2
}

BINDIR="${HOME}/.local/bin"
VERSION="latest"

while getopts "b:h" opt; do
  case "$opt" in
    b) BINDIR="$OPTARG" ;;
    h) usage ;;
    *) usage ;;
  esac
done
shift $((OPTIND-1))

if [ "${1:-}" != "" ]; then
  VERSION="$1"
fi

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "unsupported arch: $ARCH" >&2; exit 1 ;;
esac

case "$OS" in
  darwin|linux) ;;
  *) echo "unsupported os: $OS" >&2; exit 1 ;;
esac

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

ASSET="${BIN}_${OS}_${ARCH}.tar.gz"
if [ "$VERSION" = "latest" ]; then
  URL="${BASE_URL}/latest/download/${ASSET}"
  SUMURL="${BASE_URL}/latest/download/checksums.txt"
else
  URL="${BASE_URL}/download/${VERSION}/${ASSET}"
  SUMURL="${BASE_URL}/download/${VERSION}/checksums.txt"
fi

echo "Downloading: $URL"
curl -fsSL "$URL" -o "$TMP/$ASSET"
curl -fsSL "$SUMURL" -o "$TMP/checksums.txt"

cd "$TMP"

if command -v shasum >/dev/null 2>&1; then
  EXPECTED="$(grep " $ASSET\$" checksums.txt | awk '{print $1}')"
  ACTUAL="$(shasum -a 256 "$ASSET" | awk '{print $1}')"
elif command -v sha256sum >/dev/null 2>&1; then
  EXPECTED="$(grep " $ASSET\$" checksums.txt | awk '{print $1}')"
  ACTUAL="$(sha256sum "$ASSET" | awk '{print $1}')"
else
  echo "missing shasum/sha256sum for checksum verification" >&2
  exit 1
fi

if [ "$EXPECTED" = "" ] || [ "$EXPECTED" != "$ACTUAL" ]; then
  echo "checksum mismatch" >&2
  exit 1
fi

tar -xzf "$ASSET"
mkdir -p "$BINDIR"
install -m 0755 "$BIN" "$BINDIR/$BIN"

DATA_DIR="${HOME}/.local/share/ackchyually"
SHIM_DIR="${DATA_DIR}/shims"
mkdir -p "$SHIM_DIR"

say ""
say "${GREEN}${BOLD}Installed${RESET} ${BOLD}${BIN}${RESET}"
say "  bin:   $BINDIR/$BIN"
say "  shims: $SHIM_DIR"

case ":$PATH:" in
  *":$BINDIR:"*) ;;
  *)
    say ""
    say "${YELLOW}${BOLD}To run ${BIN} now, add it to PATH:${RESET}"
    say "  export PATH=\"$BINDIR:\$PATH\""
    ;;
esac

say ""
say "${BOLD}Next (required): put shims first in PATH${RESET}"
say "  export PATH=\"$SHIM_DIR:\$PATH\""
say "  # for future shells, add that line to your ~/.zshrc or ~/.bashrc"
say "  hash -r 2>/dev/null || true"
say ""
say "${BOLD}Then install shims for tools you use:${RESET}"
say "  $BIN shim install git gh xcodebuild"
say ""
say "${BOLD}Verify shims are active:${RESET}"
say "  which gh"
say "  # $SHIM_DIR/gh"
say ""
say "${DIM}If it prints something like /opt/homebrew/bin/gh, the shim dir isn't first in PATH.${RESET}"
