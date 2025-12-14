#!/bin/sh
set -eu

REPO="joelklabo/ackchyually"
BIN="ackchyually"
BASE_URL="https://github.com/${REPO}/releases"

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

echo "Installed $BIN to $BINDIR"
echo "Next: $BIN shim install git gh xcodebuild"

