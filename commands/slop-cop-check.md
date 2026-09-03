---
name: slop-cop-check
description: "Run slop-cop over a file or the current selection and show the violations report. Usage: /slop-cop-check [path]"
---

# Slop Cop: manual check

Run `slop-cop check` over the target prose and summarise the findings. Does
*not* rewrite the text. For a self-review + revise loop, rely on the
`slop-cop-prose` skill instead.

## Usage

- `/slop-cop-check`: check the open file or the current selection.
- `/slop-cop-check <path>`: check the file at `<path>`.
- `/slop-cop-check -`: read the target text from stdin (useful for piping).

## Instructions

1. **Resolve the binary.** Use
   `${CLAUDE_PLUGIN_ROOT:-${CURSOR_PLUGIN_ROOT:-}}/bin/slop-cop`, the committed
   wrapper that execs the release pinned in `bin/slop-cop.binrun`, and fall
   back to `slop-cop` on `$PATH` when neither plugin root is set. Verify it
   with `"$SLOP_COP" --version`: one bare version line. A binary that answers
   `unknown flag: --version` is stale; discard it and resolve again. Never
   take a binary from a plugin data directory such as
   `~/.claude/plugins/data/<plugin>/bin`, and never hardcode a discovered
   absolute path into a later command or a memory note; that includes the
   report's `binary_path`.

2. **Pick the target.** In order of preference:
   - `$ARGUMENTS` if the user supplied a path or `-`.
   - The current editor selection if any is non-empty.
   - The focused file otherwise.

3. **Invoke.** Run `"$SLOP_COP" check --format=compact <target>`. slop-cop
   picks the input language from the extension (`.md`, `.html`, `.jsx`,
   `.tsx`, `.ts`, `.js`) and masks non-prose regions before running
   detectors. Pass `--lang=<mode>` to override, such as `--lang=text` on
   a code file to see every regex hit, or `--lang=markdown` when reading
   from stdin. Add `--no-llm` when the user wants the answer in under a
   second; the sentence tier adds about 10s.

4. **Confirm it ran.** Compact output ends in a `counts` line, and a clean
   run is that line alone; no `counts` line, or empty output, means a killed
   run to repeat, never clean text.

5. **Summarise.** Present each `ruleId<TAB>line:col<TAB>matchedText` line as
   a short bullet, then the `counts` totals. For a rule ID that needs
   explaining, re-run without `--format=compact` and read the `rules`
   sidecar's `tip`, plus any `explanation` or `suggestedChange` on the
   violation. Close with: "Run `/slop-cop-check` again after edits, or ask
   for a revision through the `slop-cop-prose` skill."

Do **not** rewrite the file from this command. This is a report-only tool.
