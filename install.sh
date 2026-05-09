#!/bin/sh
set -eu

unset GREP_OPTIONS
export LC_ALL=C

REPO="imbanytuidoter/base-node-helper"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
COSIGN_OIDC_ISSUER="https://token.actions.githubusercontent.com"
COSIGN_IDENTITY_RE="https://github.com/imbanytuidoter/base-node-helper/.github/workflows/release.yml@refs/tags/v"
ALLOW_NO_COSIGN="${ALLOW_NO_COSIGN:-0}"

# Parse flags
for arg in "$@"; do
  case "$arg" in
    --allow-no-cosign) ALLOW_NO_COSIGN=1 ;;
    *) echo "unknown flag: $arg" >&2; exit 1 ;;
  esac
done

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

# [MED] input-validation: explicitly allow-list both OS and ARCH values.
case "$OS" in
  linux|darwin) ;;
  *) echo "ERROR: unsupported OS: $OS" >&2; exit 1 ;;
esac
case "$ARCH" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) echo "ERROR: unsupported arch: $ARCH" >&2; exit 1 ;;
esac

if command -v jq >/dev/null 2>&1; then
  VERSION="${VERSION:-$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | jq -r '.tag_name')}"
else
  VERSION="${VERSION:-$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | head -1 | cut -d'"' -f4)}"
fi
[ -n "$VERSION" ] || { echo "ERROR: could not detect latest version" >&2; exit 1; }

# [MED] input-validation: strict semver check — reject anything with slashes,
# double-dots, or non-numeric characters beyond the semver format.
case "$VERSION" in
  v[0-9]*.[0-9]*.[0-9]*)
    case "$VERSION" in
      */*|*..*) echo "ERROR: suspicious version string: $VERSION" >&2; exit 1 ;;
    esac ;;
  *) echo "ERROR: unexpected version format: $VERSION" >&2; exit 1 ;;
esac

# Construct all URLs from validated, constrained components only.
ARCHIVE="bnh_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"
SUMS_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"
SIG_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt.sig"
CERT_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt.pem"

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

echo "Downloading bnh ${VERSION} (${OS}/${ARCH})..."
curl -fsSL -o "$TMP/bnh.tar.gz"      "$URL"
curl -fsSL -o "$TMP/checksums.txt"   "$SUMS_URL"

# --- cosign signature verification (supply-chain protection) ---
if command -v cosign >/dev/null 2>&1; then
  curl -fsSL -o "$TMP/checksums.txt.sig"  "$SIG_URL"
  curl -fsSL -o "$TMP/checksums.txt.pem"  "$CERT_URL"
  cosign verify-blob \
    --certificate         "$TMP/checksums.txt.pem" \
    --signature           "$TMP/checksums.txt.sig" \
    --certificate-identity-regexp "$COSIGN_IDENTITY_RE" \
    --certificate-oidc-issuer     "$COSIGN_OIDC_ISSUER" \
    "$TMP/checksums.txt" \
    || { echo "ERROR: cosign signature verification failed — aborting" >&2; exit 1; }
  echo "OK: cosign signature verified"
else
  # [LOW] insecure-default: cosign is required by default. Pass --allow-no-cosign
  # only if you have independently verified the binary through another channel.
  if [ "$ALLOW_NO_COSIGN" != "1" ]; then
    echo "ERROR: cosign not found. Install it for full supply-chain protection:" >&2
    echo "       https://docs.sigstore.dev/cosign/system_config/installation/" >&2
    echo "" >&2
    echo "To skip signature verification (NOT recommended):" >&2
    echo "  ALLOW_NO_COSIGN=1 $0 --allow-no-cosign" >&2
    exit 1
  fi
  echo "WARNING: cosign signature verification skipped (--allow-no-cosign)" >&2
fi

# --- SHA-256 checksum verification ---
# [MED] input-validation: double-quote ${ARCHIVE} in grep to prevent
# metacharacter injection from the (allow-listed) archive name.
EXPECTED=$(grep -- "${ARCHIVE}" "$TMP/checksums.txt" | awk '{print $1}')
[ -n "$EXPECTED" ] || { echo "ERROR: ${ARCHIVE} not found in checksums.txt" >&2; exit 1; }
ACTUAL=$(shasum -a 256 "$TMP/bnh.tar.gz" 2>/dev/null | cut -d' ' -f1 \
         || sha256sum "$TMP/bnh.tar.gz" | cut -d' ' -f1)
[ "$EXPECTED" = "$ACTUAL" ] || { echo "ERROR: checksum mismatch — aborting" >&2; exit 1; }
echo "OK: checksum verified"

tar -xzf "$TMP/bnh.tar.gz" -C "$TMP"
install -m 755 "$TMP/bnh" "$INSTALL_DIR/bnh"
echo "Installed: $INSTALL_DIR/bnh"
"$INSTALL_DIR/bnh" version
