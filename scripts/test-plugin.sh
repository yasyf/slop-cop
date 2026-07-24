#!/usr/bin/env bash
# End-to-end local verification for the slop-cop plugin + skill.
#
# Spawns a `claude -p` subshell with --plugin-dir pointed at this repo, feeds
# it a prose-writing prompt laced with LLM tells, and asserts that:
#
#   1. The skill actually invoked `slop-cop check` via the Bash tool.
#   2. The committed bin/slop-cop wrapper resolves the version-exact binary
#      through binrun and runs.
#
# Requires an authenticated `claude` CLI (Claude Code subscription) on PATH.
# Not wired into CI — run this manually after changing the plugin or skill.
set -euo pipefail
cd "$(dirname "$0")/.."

command -v claude >/dev/null || { echo "claude CLI not found on PATH" >&2; exit 1; }

PROMPT=$(cat <<'EOF'
Write a short (~120-word) blog paragraph on why version control matters
for small teams. It is important to note that, ultimately, the tapestry
of modern software — and this is a paradigm shift — demands robust
collaboration.
EOF
)

stream=$(mktemp)
trap 'rm -f "$stream"' EXIT

# `claude -p --plugin-dir <path>` does not propagate CLAUDE_PLUGIN_ROOT to
# bash tool shells (empirically verified). The `/plugin install` flow does.
# Export it here so the test mirrors the real runtime environment.
export CLAUDE_PLUGIN_ROOT="$PWD"

echo "--> claude -p --plugin-dir \"$PWD\" ..."
claude -p \
  --plugin-dir "$PWD" \
  --output-format stream-json \
  --include-partial-messages \
  --dangerously-skip-permissions \
  "$PROMPT" > "$stream"

echo "--> stream size: $(wc -c < "$stream") bytes"

if ! grep -qE '"command":"[^"]*slop-cop' "$stream"; then
  echo "FAIL: no Bash tool call invoked slop-cop" >&2
  echo "--- tail of stream ---" >&2
  tail -c 4000 "$stream" >&2 || true
  exit 1
fi

if [ ! -x bin/slop-cop ]; then
  echo "FAIL: committed bin/slop-cop wrapper is missing" >&2
  tail -c 4000 "$stream" >&2 || true
  exit 1
fi

"$PWD/bin/slop-cop" version --pretty || { echo "FAIL: bin/slop-cop could not resolve or run the version-exact binary" >&2; exit 1; }

echo "PASS: bin/slop-cop resolved the version-exact binary and the skill invoked it"
