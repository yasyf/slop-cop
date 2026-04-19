---
name: slop-cop-prose
description: Detects and fixes LLM-generated prose tells using the slop-cop CLI. Triggers whenever the user asks the agent to write, draft, revise, polish, or edit natural-language prose (blog posts, docs, PR descriptions, commit messages, release notes, marketing copy, emails). The agent pipes its own draft through `slop-cop check -`, revises based on the JSON violation report, and delivers only the cleaned result.
allowed-tools: Bash(slop-cop:*), Bash(bash:*), Read
---

# Slop Cop (prose)

Before returning a piece of prose to the user (a blog paragraph, a doc, a PR
description, a commit message, a release note, an email), run it through
`slop-cop` to catch LLM-generated writing tells (overused intensifiers,
filler adverbs, negation pivots, em-dash abuse, throat-clearing, hedge
stacks, metaphor crutches, and so on), and revise.

This is a **self-review** loop: the draft is yours, the review tool is also
yours, and the user sees only the revised result.

## When to run

Run this skill whenever the user asks to:

- Write or draft prose (blog posts, docs, marketing copy, summaries, emails).
- Revise, polish, edit, or shorten existing prose.
- Produce a PR description, commit message, changelog entry, or release notes.

Do **not** run it on:

- Code, SQL, JSON, YAML, configs, shell commands, or other non-prose artefacts.
- Single-sentence acknowledgements (e.g. "Done.", "Pushed.").
- Content the user explicitly wants preserved verbatim.

## Resolving the binary

Before every run of the skill, evaluate these three options in order and
use the first one that works:

```bash
# 1. Pre-installed on PATH (CI, scripting, or the user already has it).
if command -v slop-cop >/dev/null 2>&1; then
  SLOP_COP=slop-cop
# 2. Bundled inside the plugin (normal install path).
elif [ -x "${CLAUDE_PLUGIN_ROOT:-${CURSOR_PLUGIN_ROOT:-}}/bin/slop-cop" ]; then
  SLOP_COP="${CLAUDE_PLUGIN_ROOT:-${CURSOR_PLUGIN_ROOT}}/bin/slop-cop"
# 3. First run: bootstrap the binary into the plugin dir, then use it.
else
  PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-${CURSOR_PLUGIN_ROOT:-}}"
  bash "${PLUGIN_ROOT}/scripts/install-binary.sh"
  SLOP_COP="${PLUGIN_ROOT}/bin/slop-cop"
fi
```

On Windows replace the final step with
`pwsh "$env:CLAUDE_PLUGIN_ROOT\scripts\install-binary.ps1"` (or
`powershell -File ...`) and point `SLOP_COP` at `bin\slop-cop.exe`.

The installer is idempotent (a no-op when the binary is already present),
so calling it on every skill invocation is safe but wasteful. Prefer the
pre-check above.

If *both* `CLAUDE_PLUGIN_ROOT` and `CURSOR_PLUGIN_ROOT` are unset and
`slop-cop` is not on `$PATH` (rare: running the skill outside both products
and without a prior install), infer the plugin root from this SKILL.md's
location: the plugin root is the directory two levels above this file
(from `skills/slop-cop-prose/SKILL.md`, that's the repo root). Then run
`bash <plugin_root>/scripts/install-binary.sh` the same way.

## Loop

1. **Draft.** Write the prose the user asked for.

2. **Check.** Pipe the draft on stdin:
   ```bash
   printf '%s' "$DRAFT" | "$SLOP_COP" check --markdown -
   ```
   `slop-cop` prints a JSON document of shape
   `{"text_length": N, "violations": [...], "counts_by_rule": {...}, "counts_by_category": {...}, "markdown": true, "llm": {...}}`.

   Always pass `--markdown`. LLM drafts are typically markdown-shaped (code
   fences, inline code, links, headings), and markdown mode masks those
   non-prose regions so detectors only see the actual writing. It is a
   no-op on plain prose. When checking a file path, `slop-cop check
   article.md` already auto-activates markdown mode by extension.

   Running inside Claude Code or Cursor, slop-cop detects the plugin
   environment (`$CLAUDE_PLUGIN_ROOT` / `$CURSOR_PLUGIN_ROOT`) and the
   `claude` CLI and defaults `--llm-effort=auto` → `high` (both sentence
   and document tiers). The `llm_effort` + `llm` fields in the JSON
   report tell you what actually ran; if `claude` isn't reachable, those
   passes are reported as skipped with an `error` message and the
   client-side results are still returned. Pass `--llm-effort=off` (or
   `--llm-effort=low` for sentence-only) if you want to cut cost or speed
   for small edits.

3. **Revise.** Walk the `violations` array, prioritising these high-signal
   rules first. The canonical fix for each:

   | Rule ID                  | Fix                                                                                    |
   | ------------------------ | -------------------------------------------------------------------------------------- |
   | `elevated-register`      | Replace `utilize` with `use`, `commence` with `start`, `facilitate` with `help`, `demonstrate` with `show`. |
   | `filler-adverbs`         | Delete sentence-opening `importantly`, `essentially`, `fundamentally`, `ultimately`. |
   | `hedge-stack`            | Keep at most one hedge per sentence; commit to the claim.                              |
   | `em-dash-pivot`          | Replace the em-dash with the right punctuation (comma, colon, period, parentheses).    |
   | `negation-pivot`         | Rewrite `not X, but Y` as a direct positive claim.                                     |
   | `metaphor-crutch`        | Cut clichés like `north star`, `game changer`, `deep dive`, `paradigm shift`; say the thing plainly. |
   | `important-to-note`      | Delete the phrase; just say the thing.                                                 |
   | `throat-clearing`        | Delete the preamble paragraph entirely.                                                |
   | `sycophantic-frame`      | Delete the compliment.                                                                 |

   Each violation's `matchedText` tells you exactly what to change. On
   LLM-backed rules (the `--llm` / `--llm-deep` tiers), `suggestedChange`
   may propose a replacement; use it when present. For client-side rules,
   apply the canonical fix from the table above.

4. **Loop.** Re-run `slop-cop check -` on the revised draft. Stop when
   `counts_by_rule` is empty *or* the only remaining hits are intentional
   stylistic choices you can justify. Two to three passes usually suffices.

5. **Deliver.** Return the revised prose to the user. Do not paste the JSON
   report unless the user explicitly asks for it. Do not announce the loop
   ("I ran it through slop-cop…"); the point is that the result reads
   clean, not that the process happened.

## Worked example

Draft the agent wrote (deliberately sloppy, to demonstrate what the skill
catches):

> In an era of rapid change, it is important to note that, ultimately, the
> tapestry of modern software — and this is a paradigm shift — demands
> robust collaboration.

`slop-cop check -` flags: `era-opener`, `important-to-note`, `filler-adverbs`
(ultimately), `overused-intensifiers` (tapestry, paradigm, robust),
`metaphor-crutch` (paradigm shift), `em-dash-pivot`.

Revision:

> Modern software is built by teams, and teams need version control to stay
> sane.

Second pass: `counts_by_rule: {}`. Done. That's what the user sees.

## Semantic tiers

Two LLM passes layer on top of the 35 client-side detectors, selected via
`--llm-effort=off|low|high|auto`. `auto` is the default and resolves to
`high` under the plugin, so usually you don't need to think about it.

| Effort | Passes run                                 | Extra rules caught                               |
| ------ | ------------------------------------------ | ------------------------------------------------ |
| `off`  | none                                       | —                                                |
| `low`  | sentence tier (Claude Haiku)               | `balanced-take`, `unnecessary-elaboration`, `grandiose-stakes`, `empathy-performance`, `sycophantic-frame`, `throat-clearing`, `pivot-paragraph`, `historical-analogy`, `false-vulnerability`, `triple-construction` |
| `high` | sentence + document (Haiku + Sonnet)       | the low list + `dead-metaphor`, `one-point-dilution`, `fractal-summaries` |

Sugar aliases: `--llm` ≡ `--llm-effort=low`, `--llm-deep` ≡ `--llm-effort=high`.

Both tiers shell out to `claude -p --output-format json --json-schema ...`.
If the `claude` CLI is missing or fails (no auth, rate limit, timeout),
the auto-enabled pass is skipped rather than erroring; the client-side
detector output is always returned. Inspect `llm_effort` and
`llm.sentence` / `llm.document` in the JSON report to see what actually
ran.
