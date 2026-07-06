#!/bin/sh
# canonical: cc-skills/plugins/repo-bootstrap@ed090c97704f984ff2caed8f9f224816237fe468
# Provision the slop-cop binary for the slop-cop plugin.
#
# bin/slop-cop is only ever a symlink — to a brew-installed binary, the
# durable download under the plugin data dir, or a local dev build. The payload
# never lives in the plugin dir itself: CLAUDE_PLUGIN_ROOT is ephemeral (the
# path swaps on every plugin update), while the data dir survives updates.
#
# Resolution order (semantic — reordering resurrects the dev-clobber bug):
#   1. symlink already resolves to the target release -> done
#   2. symlink resolves to a dev build                -> leave it alone
#   3. binary on PATH (brew-owned or otherwise)       -> symlink it, brew upgrade when stale
#   4. brew present, binary absent                    -> brew install, symlink
#   5. durable data-dir payload at target             -> re-symlink, no re-download
#   6. static download into the data dir              -> sha256-verify, symlink
set -eu

NAME="slop-cop"
REPO="yasyf/slop-cop"
BREW_PKG="yasyf/tap/slop-cop"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
LINK="$ROOT/bin/$NAME"
# CLAUDE_PLUGIN_DATA is only exported to hook/MCP subprocesses; bare shell runs
# fall back to its documented default.
DATA_DIR="${CLAUDE_PLUGIN_DATA:-$HOME/.claude/plugins/data/slop-cop}"

# Latest mode: resolve the newest release tag off the releases/latest redirect.
# Unresolvable (offline) with a working binary in place -> keep what we have.
effective="$(curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/$REPO/releases/latest" || true)"
TAG="${effective##*/tag/}"
case "$TAG" in
  v[0-9]*) ;;
  *)
    if [ -x "$LINK" ] && "$LINK" --version >/dev/null 2>&1; then
      exit 0
    fi
    echo "$NAME: could not resolve the latest $REPO release (got '$effective')" >&2
    exit 1
    ;;
esac

# Version output is compared v-stripped: goreleaser release binaries print the
# bare tag (v0.5.0) while brew formula builds stamp their own ldflags (0.5.0).
BARE="${TAG#v}"

# Arms 1+2: exact target exits, a dev build (describe/pseudo-version suffix, or
# the bare "dev" of an unstamped build) is never clobbered, a stale release or
# no version output falls through.
if [ -x "$LINK" ]; then
  case "$("$LINK" --version 2>/dev/null | head -n 1)" in
    "$TAG" | "$BARE") exit 0 ;;
    dev | v[0-9]*[!0-9.]* | [0-9]*[!0-9.]*) exit 0 ;;
    v[0-9]* | [0-9]*) ;;
    *) ;;
  esac
fi

mkdir -p "$ROOT/bin"

# Arm 3: a binary already on PATH wins (brew is authoritative even when it
# trails the pin). Exclude bin/ or the probe finds the managed symlink itself;
# entries are compared by inode (-ef), or a non-canonical spelling like
# "$ROOT/bin/", "$ROOT/bin/." or "$ROOT/bin//" evades the exclusion and $LINK
# becomes a self-loop.
# probe: PATH="$ROOT/bin/.:$PATH" (or bin/ or bin//) with a stale executable at
# bin/$NAME -> must never symlink bin/$NAME to itself; resolve elsewhere or
# fall through.
probe_path=
IFS_SAVE="$IFS"
IFS=:
for dir in $PATH; do
  if [ -n "$dir" ] && ! [ "$dir" -ef "$ROOT/bin" ]; then
    probe_path="$probe_path$dir:"
  fi
done
IFS="$IFS_SAVE"
probe() {
  (
    PATH="${probe_path%:}"
    command -v "$NAME" 2>/dev/null
  ) || true
}

found="$(probe)"
# Belt for the exclusion above: a probe result that is bin/$NAME's own
# directory entry under any path spelling must never be self-symlinked. The
# compare is on the parent directory's inode, not the resolved file — $LINK
# legitimately points at a stale brew binary the upgrade path must still see.
if [ -n "$found" ] && [ "$(dirname "$found")" -ef "$ROOT/bin" ]; then
  found=""
fi
if [ -n "$found" ]; then
  case "$("$found" --version 2>/dev/null | head -n 1)" in
    "$TAG" | "$BARE" | v[0-9]*[!0-9.]* | [0-9]*[!0-9.]*) ;;
    *) brew upgrade "$BREW_PKG" >/dev/null 2>&1 || true ;;
  esac
  ln -sf "$found" "$LINK"
  exit 0
fi

# Arm 4: brew present, binary absent. Best-effort — any failure (Homebrew
# tap-trust sandbox bug #22603, network) falls through to the direct download.
if command -v brew >/dev/null 2>&1; then
  if brew install "$BREW_PKG" >/dev/null 2>&1; then
    found="$(probe)"
    if [ -n "$found" ]; then
      ln -sf "$found" "$LINK"
      exit 0
    fi
  fi
  echo "$NAME: Homebrew unavailable or failed (e.g. tap-trust #22603); using direct download" >&2
fi

# Arms 5+6: the durable data dir — reuse a payload already at the target
# release, else download the bare per-platform release binary, verified
# against goreleaser's checksums.txt.
os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$os" in
  darwin | linux) ;;
  *)
    echo "$NAME: unsupported OS '$os'" >&2
    exit 1
    ;;
esac
arch="$(uname -m)"
case "$arch" in
  x86_64 | amd64) arch=amd64 ;;
  arm64 | aarch64) arch=arm64 ;;
  *)
    echo "$NAME: unsupported architecture '$arch'" >&2
    exit 1
    ;;
esac

sha256_of() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

asset="${NAME}_${os}_${arch}"
url="https://github.com/$REPO/releases/download/$TAG/$asset"
dest="$DATA_DIR/bin/$NAME"

# A durable payload already at the target release is reused: post-update
# reprovisioning (the plugin root swaps, the gitignored symlink is gone) is a
# symlink repair, not a re-download. A stale payload falls through and is
# overwritten by the download below.
# probe: seed $dest with a stub printing "$TAG", rm bin/$NAME, run with a
# brew-less curl-less PATH -> must symlink $dest and exit 0 without a fetch.
if [ -x "$dest" ]; then
  case "$("$dest" --version 2>/dev/null | head -n 1)" in
    "$TAG" | "$BARE")
      ln -sf "$dest" "$LINK"
      exit 0
      ;;
  esac
fi

echo "$NAME: downloading $url" >&2
mkdir -p "$DATA_DIR/bin"
# Stage on the destination filesystem and rename into place: writing onto a
# running executable fails with ETXTBSY on Linux, and the rename keeps any
# still-executing inode alive.
tmp="$(mktemp "$DATA_DIR/bin/.$NAME.XXXXXX")"
trap 'rm -f "$tmp"' EXIT
curl -fsSL --retry 2 -o "$tmp" "$url"

if ! sums="$(curl -fsSL --retry 2 "https://github.com/$REPO/releases/download/$TAG/checksums.txt")"; then
  echo "$NAME: could not fetch checksums.txt for $TAG" >&2
  exit 1
fi
expected="$(printf '%s\n' "$sums" | awk -v a="$asset" '$2 == a {print $1}')"
if [ -z "$expected" ]; then
  echo "$NAME: no checksum for $asset in checksums.txt" >&2
  exit 1
fi
actual="$(sha256_of "$tmp")"
if [ "$actual" != "$expected" ]; then
  echo "$NAME: checksum mismatch for $asset (expected $expected, got $actual)" >&2
  exit 1
fi

chmod +x "$tmp"
mv -f "$tmp" "$dest"
ln -sf "$dest" "$LINK"
echo "$NAME: installed $dest ($("$dest" --version))" >&2
