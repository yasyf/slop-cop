#!/usr/bin/env bash
# Developer ID code-sign + notarize one macOS build artifact.
#
# Invoked per-build by goreleaser's `builds.hooks.post`:
#     scripts/macos-codesign.sh "{{ .Path }}" "{{ .Target }}"
#
# Why not goreleaser's built-in `notarize`? It signs with quill, whose arm64
# signatures get SIGKILLed by macOS 15/26 at exec (anchore/quill#566, closed
# "not planned"). So we sign with Apple's own codesign + notarytool on the macOS
# runner instead. No-op unless the target is darwin AND the signing env is set.
#
# Env (set by .github/workflows/release.yml after importing the cert):
#   MACOS_SIGN_IDENTITY   — "Developer ID Application: NAME (TEAMID)"
#   MACOS_NOTARY_KEY_FILE — path to the App Store Connect API .p8 (enables notarization)
#   MACOS_NOTARY_KEY_ID   — the key's Key ID
#   MACOS_NOTARY_ISSUER   — the team's Issuer ID
set -euo pipefail

bin=$1
target=${2:-}

case "$target" in
  darwin_*) ;;
  *) exit 0 ;;
esac

if [ -z "${MACOS_SIGN_IDENTITY:-}" ]; then
  echo "macos-codesign: MACOS_SIGN_IDENTITY unset — leaving $bin unsigned"
  exit 0
fi

echo "macos-codesign: signing $bin ($target)"
codesign --force --options runtime --timestamp -s "$MACOS_SIGN_IDENTITY" "$bin"
codesign --verify --strict --verbose=2 "$bin"

if [ -n "${MACOS_NOTARY_KEY_FILE:-}" ]; then
  echo "macos-codesign: notarizing $bin"
  zip="$(mktemp -d)/$(basename "$bin").zip"
  ditto -c -k "$bin" "$zip"
  xcrun notarytool submit "$zip" \
    --key "$MACOS_NOTARY_KEY_FILE" \
    --key-id "$MACOS_NOTARY_KEY_ID" \
    --issuer "$MACOS_NOTARY_ISSUER" \
    --wait --timeout 20m
  # A bare Mach-O can't be stapled; notarization is recorded against its cdhash
  # and verified online by Gatekeeper.
fi
