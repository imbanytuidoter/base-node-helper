#!/bin/sh
set -eu

unset GREP_OPTIONS
export LC_ALL=C

REPO="base-helper/base-node-helper"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) echo "unsupported arch: $ARCH" >&2; exit 1 ;;
esac

if command -v jq >/dev/null 2>&1; then
  VERSION="${VERSION:-$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | jq -r '.tag_name')}"
else
  VERSION="${VERSION:-$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | head -1 | cut -d'"' -f4)}"
fi
[ -n "$VERSION" ] || { echo "could not detect latest version"; exit 1; }
case "$VERSION" in
  v[0-9]*.[0-9]*.[0-9]*) ;;
  *) echo "ERROR: unexpected version format: $VERSION" >&2; exit 1 ;;
esac

URL="https://github.com/${REPO}/releases/download/${VERSION}/base-node-helper_${OS}_${ARCH}.tar.gz"
SUMS_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

echo "downloading $URL"
curl -fsSL -o "$TMP/bnh.tar.gz" "$URL"
curl -fsSL -o "$TMP/checksums.txt" "$SUMS_URL"

EXPECTED=$(grep "base-node-helper_${OS}_${ARCH}.tar.gz" "$TMP/checksums.txt" | cut -d' ' -f1)
ACTUAL=$(shasum -a 256 "$TMP/bnh.tar.gz" 2>/dev/null | cut -d' ' -f1 || sha256sum "$TMP/bnh.tar.gz" | cut -d' ' -f1)
[ "$EXPECTED" = "$ACTUAL" ] || { echo "checksum mismatch: $ACTUAL != $EXPECTED" >&2; exit 1; }

tar -xzf "$TMP/bnh.tar.gz" -C "$TMP"
install -m 755 "$TMP/base-node-helper" "$INSTALL_DIR/base-node-helper"
echo "installed: $INSTALL_DIR/base-node-helper"
"$INSTALL_DIR/base-node-helper" version
