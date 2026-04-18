---
name: slop-cop-check
description: "Run slop-cop over a file or the current selection and show the JSON violations report. Usage: /slop-cop-check [path]"
---

# Slop Cop — manual check

Run `slop-cop check` over the target prose and summarise the findings. Does
*not* rewrite the text — for a self-review + revise loop, rely on the
`slop-cop-prose` skill instead.

## Usage

- `/slop-cop-check` — check the currently open file or the current selection.
- `/slop-cop-check <path>` — check the file at `<path>`.
- `/slop-cop-check -` — read the target text from stdin (useful for piping).

## Instructions

1. **Resolve the binary.** Prefer
   `${CLAUDE_PLUGIN_ROOT:-${CURSOR_PLUGIN_ROOT:-}}/bin/slop-cop`. If it is
   not present, run the bootstrap once:
   ```bash
   bash "${CLAUDE_PLUGIN_ROOT:-${CURSOR_PLUGIN_ROOT}}/scripts/install-binary.sh"
   ```
   (`install-binary.ps1` on Windows). Fall back to `slop-cop` on `$PATH` if
   neither plugin root is set.

2. **Pick the target.** In order of preference:
   - `$ARGUMENTS` if the user supplied a path or `-`.
   - The current editor selection if any is non-empty.
   - The currently focused file otherwise.

3. **Invoke.** Run `slop-cop check --pretty <target>`.

4. **Summarise.** Parse the JSON and present:
   - The total count and `counts_by_category` breakdown.
   - Each violation as a short bullet: rule ID, the matched span
     (first ~60 chars), and (if present) the `explanation` / `suggestedChange`.
   - A final pointer: "Run `/slop-cop-check` again after edits, or ask for a
     revision via the `slop-cop-prose` skill."

Do **not** rewrite the file from this command. This is a report-only tool.
