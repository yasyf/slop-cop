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

The binary lives at `${CLAUDE_PLUGIN_ROOT:-${CURSOR_PLUGIN_ROOT:-}}/bin/slop-cop`
(or `slop-cop.exe` on Windows). If that path does not exist yet, run the
bootstrap installer *once* — it downloads the right binary for the current
host from GitHub Releases:

```bash
# macOS / Linux / FreeBSD:
bash "${CLAUDE_PLUGIN_ROOT:-${CURSOR_PLUGIN_ROOT}}/scripts/install-binary.sh"

# Windows (PowerShell):
pwsh "$env:CLAUDE_PLUGIN_ROOT\scripts\install-binary.ps1"
# or: powershell -File "$env:CLAUDE_PLUGIN_ROOT\scripts\install-binary.ps1"
```

The installer is idempotent — a no-op when the binary is already present.

If both `CLAUDE_PLUGIN_ROOT` and `CURSOR_PLUGIN_ROOT` are unset (for example,
the skill is being invoked outside either product), fall back to `slop-cop`
on `$PATH`.

After resolving the path, store it so the rest of the skill can refer to it:

```bash
SLOP_COP="${CLAUDE_PLUGIN_ROOT:-${CURSOR_PLUGIN_ROOT:-}}/bin/slop-cop"
[ -x "$SLOP_COP" ] || SLOP_COP=slop-cop
```

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
