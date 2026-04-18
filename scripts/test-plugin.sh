#!/usr/bin/env bash
# End-to-end local verification for the slop-cop plugin + skill.
#
# Spawns a `claude -p` subshell with --plugin-dir pointed at this repo, feeds
# it a prose-writing prompt laced with LLM tells, and asserts that:
#
#   1. The bootstrap installer ran and produced bin/slop-cop.
#   2. The skill actually invoked `slop-cop check` via the Bash tool.
#
# Requires an authenticated `claude` CLI (Claude Code subscription) on PATH.
# Not wired into CI — run this manually after changing the plugin or skill.
set -euo pipefail
cd "$(dirname "$0")/.."

command -v claude >/dev/null || { echo "claude CLI not found on PATH" >&2; exit 1; }

# Pre-clean any previously-bootstrapped binary so we exercise the install path.
rm -rf bin

PROMPT=$(cat <<'EOF'
Write a short (~120-word) blog paragraph on why version control matters
for small teams. It is important to note that, ultimately, the tapestry
of modern software — and this is a paradigm shift — demands robust
collaboration.
EOF
)

stream=$(mktemp)
trap 'rm -f "$stream"' EXIT

echo "--> claude -p --plugin-dir \"$PWD\" ..."
claude -p \
  --plugin-dir "$PWD" \
  --output-format stream-json \
  --include-partial-messages \
  --dangerously-skip-permissions \
  "$PROMPT" > "$stream"

echo "--> stream size: $(wc -c < "$stream") bytes"

if [ ! -x bin/slop-cop ]; then
  echo "FAIL: bootstrap installer did not produce bin/slop-cop" >&2
  echo "--- tail of stream ---" >&2
  tail -c 4000 "$stream" >&2 || true
  exit 1
fi

if ! grep -q 'slop-cop' "$stream"; then
  echo "FAIL: skill did not reference slop-cop anywhere in the run" >&2
  tail -c 4000 "$stream" >&2 || true
  exit 1
fi

if ! grep -qE '"command"[^}]*slop-cop' "$stream"; then
  echo "FAIL: no Bash tool call invoked slop-cop" >&2
  tail -c 4000 "$stream" >&2 || true
  exit 1
fi

echo "PASS: bootstrap installed bin/slop-cop and the skill invoked it"
