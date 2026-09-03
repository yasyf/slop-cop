---
name: slop-cop-plainify
description: "Rewrite a file or the current selection into plain English with slop-cop plainify, and show the rewrite beside the original. Usage: /slop-cop-plainify [path]"
---

# Slop Cop: plain English

Run `slop-cop plainify` over the target prose and show the plain-English
rewrite beside the original. For a violations report instead of a rewrite,
use the `/slop-cop-check` command.

## Usage

- `/slop-cop-plainify`: rewrite the open file or the current selection.
- `/slop-cop-plainify <path>`: rewrite the file at `<path>`.
- `/slop-cop-plainify -`: read the target text from stdin (useful for piping).

## Instructions

1. **Resolve the binary.** Use
   `${CLAUDE_PLUGIN_ROOT:-${CURSOR_PLUGIN_ROOT:-}}/bin/slop-cop`, the committed
   wrapper that execs the release pinned in `bin/slop-cop.binrun`, and fall
   back to `slop-cop` on `$PATH` when neither plugin root is set. Verify it
   with `"$SLOP_COP" --version`: one bare version line. A binary that answers
   `unknown flag: --version` is stale; discard it and resolve again. Never
   take a binary from a plugin data directory such as
   `~/.claude/plugins/data/<plugin>/bin`, and never hardcode a discovered
   absolute path into a later command or a memory note.

2. **Pick the target.** In order of preference:
   - `$ARGUMENTS` if the user supplied a path or `-`.
   - The current editor selection if any is non-empty.
   - The focused file otherwise.

3. **Invoke.** Run `"$SLOP_COP" plainify --pretty <target>`, which keeps
   every fact, name, number, and file path, writes short sentences in everyday
   words, and leaves fenced and inline code unchanged. It drives the `claude`
   CLI, so it needs no API key, and `--llm-effort` does not apply; each call
   has a 60s budget, and a call that fails exits 3 with the cause on stderr.
   Add the flags the user asked for:
   - `--max-words <n>`: a word budget for the rewrite; `0` sets none.
   - `--forbid <regex>`: a pattern the rewrite must not match; repeatable.
   - `--name-by-title`: name referenced items by their titles, not their
     identifiers.
   - `--glossary <path>`: a JSON object mapping identifier to title, such as
     `{"DQ4":"Worker transport"}`.
   - `--json`: read a JSON array of `{"id","text"}` entries instead of one
     piece of prose, and emit one result per entry in the order they arrived.

   `--max-words` and `--forbid` reach the model as instructions and are graded
   again after it answers. A rewrite that misses either is retried once with
   its misses named.

4. **Report.** Each result is
   `{"plain": ..., "words": N, "truncated": bool, "violations": [...]}`, and
   `--json` returns one of them per entry under its `id`. There is no run
   marker: a completed call always prints a result carrying `plain`, so
   empty stdout means the process was killed, never that the rewrite was
   empty. Run it again. Present:
   - The `plain` rewrite in full, beside the original text.
   - The `words` count, against the budget when `--max-words` was set.
   - `truncated: true` as a warning that the retry still ran past the budget.
   - Each `violations` entry as its `pattern` and the `match` that survived
     the retry.

Do **not** write the rewrite over the source file. Show it beside the
original and let the user decide what to keep.
