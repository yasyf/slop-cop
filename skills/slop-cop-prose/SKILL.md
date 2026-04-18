---
name: slop-cop-prose
description: Use slop-cop to detect and fix LLM-generated prose tells in any draft the agent is about to present or write — blog posts, docs, PR descriptions, commit messages, release notes, marketing copy, emails. Trigger whenever the user asks the agent to write, draft, revise, polish, or edit natural-language prose.
allowed-tools: Read, Edit, Write, Bash
---

# Slop Cop (prose)

You are about to return a piece of prose to the user — a blog paragraph, a
doc, a PR description, a commit message, a release note, an email. Before you
present it, run it through `slop-cop` to catch LLM-generated writing tells
(overused intensifiers, filler adverbs, negation pivots, em-dash abuse,
throat-clearing, hedge stacks, metaphor crutches, etc.), and revise.

This is a **self-review** loop: the draft is yours, the review tool is also
yours, and the user sees only the revised result.

## When to run

Run this skill whenever the user asks you to:

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

The installer is idempotent — a no-op when the binary is already present,
so calling it on every skill invocation is safe but wasteful. Prefer the
pre-check above.

If *both* `CLAUDE_PLUGIN_ROOT` and `CURSOR_PLUGIN_ROOT` are unset and
`slop-cop` is not on `$PATH` (rare: running the skill outside both products
and without a prior install), infer the plugin root from this SKILL.md's
location: the plugin root is the directory two levels above this file
(`skills/slop-cop-prose/SKILL.md` → repo root). Then run
`bash <plugin_root>/scripts/install-binary.sh` the same way.

## Loop

1. **Draft.** Write the prose the user asked for.
2. **Check.** Pipe the draft on stdin:
   ```bash
   printf '%s' "$DRAFT" | "$SLOP_COP" check -
   ```
   `slop-cop` prints a JSON document of shape
   `{"text_length": N, "violations": [...], "counts_by_rule": {...}, "counts_by_category": {...}}`.
3. **Revise.** Walk the `violations` array, prioritising these high-signal
   rules first:
   - `elevated-register` — replace `utilize`→`use`, `commence`→`start`,
     `facilitate`→`help`, `demonstrate`→`show`, etc.
   - `filler-adverbs` — delete sentence-opening `importantly`,
     `essentially`, `fundamentally`, `ultimately`.
   - `hedge-stack` — keep at most one hedge per sentence; commit to the claim.
   - `em-dash-pivot` — replace `—` with the right punctuation (comma, colon,
     period, parentheses).
   - `negation-pivot` — rewrite `not X, but Y` as a direct positive claim.
   - `metaphor-crutch` — cut clichés like `north star`, `game changer`,
     `deep dive`, `paradigm shift`; say the thing plainly.
   - `important-to-note` — delete and just say the thing.
   - `throat-clearing`, `sycophantic-frame` — delete preambles and compliments.

   Each violation's `matchedText` tells you exactly what to change; its
   `suggestedChange` (when present, on LLM-backed rules) proposes a
   replacement. For client-side rules, use the rule's canonical fix above.

4. **Loop.** Re-run `slop-cop check -` on the revised draft. Stop when
   `counts_by_rule` is empty *or* the only remaining hits are intentional
   stylistic choices you can justify. Two to three passes usually suffices.

5. **Deliver.** Return the revised prose to the user. Do not paste the JSON
   report unless the user explicitly asks for it. Do not announce the loop
   ("I ran it through slop-cop…") — the point is that the result reads
   clean, not that the process happened.

## Worked example

Draft the agent wrote:

> In an era of rapid change, it is important to note that, ultimately, the
> tapestry of modern software — and this is a paradigm shift — demands
> robust collaboration.

`slop-cop check -` flags: `era-opener`, `important-to-note`, `filler-adverbs`
(ultimately), `overused-intensifiers` (tapestry, paradigm, robust),
`metaphor-crutch` (paradigm shift), `em-dash-pivot`.

Revision:

> Modern software is built by teams, and teams need version control to stay
> sane.

Second pass: `counts_by_rule: {}`. Done — that's what the user sees.

## Optional: deeper analysis

For long-form drafts, `slop-cop check --llm --llm-deep -` adds sentence-tier
(Haiku) and document-tier (Sonnet) passes that catch patterns like
`balanced-take`, `unnecessary-elaboration`, `grandiose-stakes`,
`one-point-dilution`, and `fractal-summaries`. These require the `claude`
CLI on `$PATH`, which the user already has inside Claude Code. Use only when
the basic pass is clean but the writing still feels off.
