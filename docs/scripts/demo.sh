#!/usr/bin/env bash
# Regenerates docs/assets/demo.png: a real captured run of
# `slop-cop check draft.md --pretty` against the committed fixture draft.md.
# Requires slop-cop and freeze (https://github.com/charmbracelet/freeze) on $PATH.
set -euo pipefail
cd "$(dirname "$0")"

session=$(mktemp)
trap 'rm -f "$session"' EXIT

printf '\033[1;32m$\033[0m \033[1mslop-cop check draft.md --pretty\033[0m\n' > "$session"
slop-cop check draft.md --pretty >> "$session"

freeze "$session" \
  --theme github-dark \
  --background "#0d1117" \
  --window \
  --padding 24 \
  --font.size 28 \
  --output ../assets/demo.png
