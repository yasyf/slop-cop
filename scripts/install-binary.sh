#!/usr/bin/env bash
# Fetches the prebuilt slop-cop binary matching the current host into
# ${PLUGIN_ROOT}/bin/slop-cop. Designed to be invoked idempotently by the
# slop-cop-prose skill on first use; safe to re-run.
set -euo pipefail

# Resolve plugin root from either Claude Code or Cursor env; fall back to
# this script's parent directory when invoked directly.
PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-${CURSOR_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}}"
BIN_DIR="$PLUGIN_ROOT/bin"
BIN_PATH="$BIN_DIR/slop-cop"

# Fast path: if the binary already works, we're done. The skill calls us
# liberally and we don't want to re-download on every invocation.
if [ -x "$BIN_PATH" ] && "$BIN_PATH" version >/dev/null 2>&1; then
  exit 0
fi

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
  x86_64|amd64)  arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) echo "install-binary.sh: unsupported arch: $arch" >&2; exit 1 ;;
esac
case "$os" in
  darwin|linux|freebsd) ;;
  *) echo "install-binary.sh: unsupported os: $os" >&2; exit 1 ;;
esac

tarball="slop-cop_${os}_${arch}.tar.gz"
url="https://github.com/yasyf/slop-cop/releases/download/latest/${tarball}"

mkdir -p "$BIN_DIR"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "install-binary.sh: downloading $url"
curl --fail --show-error --silent --location "$url" -o "$tmp/slop-cop.tgz"
tar -C "$tmp" -xzf "$tmp/slop-cop.tgz"

mv "$tmp/slop-cop_${os}_${arch}/slop-cop" "$BIN_PATH"
chmod +x "$BIN_PATH"

"$BIN_PATH" version >/dev/null
echo "install-binary.sh: installed $BIN_PATH"
