---
name: slop-cop-prose
description: Detects and fixes LLM-generated prose tells and plain-language clarity problems using the slop-cop CLI. Triggers whenever the user asks the agent to write, draft, revise, polish, or edit natural-language prose (blog posts, docs, PR descriptions, commit messages, release notes, marketing copy, emails). The agent pipes its own draft through `slop-cop check -`, revises based on the JSON violation report, and delivers only the cleaned result.
allowed-tools: Bash(slop-cop:*), Bash(bash:*), Read
---

# Slop Cop (prose)

Before returning a piece of prose to the user, whether a blog paragraph, a
doc, a PR description, a commit message, a release note, or an email, run it
through `slop-cop` and revise. The check covers two rule layers. The slop
layer catches LLM writing tells such as overused intensifiers, filler
adverbs, negation pivots, em-dash abuse, throat-clearing, and hedge stacks.
The base layer underneath catches plain-language clarity problems such as
40-word sentences, padded verbs, and passives that demote their actor, and
those fire on human-sounding prose just as readily.

This is a **self-review** loop. The draft and the review tool are both
yours; the user sees only the revised result.

Writing always runs on fable; never delegate the draft or the revision to a
down-routed subagent. Inherit the session model or pass `model: fable`.

## When to run

Run this skill whenever the user asks you to write or draft prose, from blog
posts and docs to marketing copy, summaries, and emails. Run it too when the
ask is to revise, polish, edit, or shorten existing prose, or to produce a
PR description, commit message, changelog entry, or release notes.

Skip it for code, SQL, JSON, YAML, configs, shell commands, and other
non-prose artifacts, for single-sentence acknowledgements such as a bare
"Done.", and for content the user explicitly wants preserved verbatim.

## Resolving the binary

`bin/slop-cop` under the plugin root is a committed wrapper, a symlink to a
shim that locates the `binrun` runner and execs the exact slop-cop release
pinned in `bin/slop-cop.binrun`, downloading and caching it on first call.
The plugin pre-warms it on session start, so it is usually already resolved.
Use the wrapper first, PATH second:

```bash
PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-${CURSOR_PLUGIN_ROOT:-}}"
# 1. The committed binrun wrapper (normal path). The first call resolves and
#    caches the pinned release; every call after is a cache hit.
if [ -x "${PLUGIN_ROOT}/bin/slop-cop" ]; then
  SLOP_COP="${PLUGIN_ROOT}/bin/slop-cop"
# 2. PATH fallback (CI, scripting, or a standalone install).
else
  SLOP_COP=slop-cop
fi
```

When both `CLAUDE_PLUGIN_ROOT` and `CURSOR_PLUGIN_ROOT` are unset, which
means the skill is running outside both products, infer the plugin root from
this SKILL.md's own location. The plugin root is the directory two levels
above this file, so from `skills/slop-cop-prose/SKILL.md` it is the repo
root. The wrapper derives its paths from its own location, and
`<plugin_root>/bin/slop-cop` works the same way there.

## Loop

1. **Draft.** Write the prose the user asked for.

2. **Check.** Pipe the draft on stdin:
   ```bash
   printf '%s' "$DRAFT" | "$SLOP_COP" check --lang=markdown -
   ```
   `slop-cop` prints a JSON document of shape
   `{"text_length": N, "violations": [...], "counts_by_rule": {...}, "counts_by_category": {...}, "lang": "markdown", "llm_effort": "high", "llm": {...}, "readability": {...}}`.
   The `llm` object appears when an LLM pass runs, and `readability` appears
   when the base layer runs over at least 100 words of prose.

   Pass `--lang=markdown` for prose drafts on stdin. LLM drafts are
   typically markdown-shaped, full of code fences, inline code, links, and
   headings, and markdown mode masks those non-prose regions so detectors
   only see the actual writing. It is safe on plain prose. When checking a
   file path, `slop-cop check article.md` picks the mode from the extension.
   Markdown extensions get markdown mode, `.html` and `.htm` get html mode,
   and `.jsx`, `.tsx`, `.ts`, and `.js` get the matching tree-sitter mode,
   which masks code so detectors only see comments, string literals,
   template quasis, and JSX text.

   With the `claude` CLI on `$PATH`, the default `--llm-effort=auto`
   resolves to `high` and runs both LLM tiers. The `llm_effort` and `llm`
   fields in the report tell you what actually ran; when `claude` is
   unreachable, slop-cop reports the auto-enabled passes as skipped with an
   `error` message and still returns the client-side results. Pass
   `--llm-effort=off`, or `--llm-effort=low` for the sentence tier alone, to
   cut cost or latency on small edits. Pass `--standard=slop` or
   `--standard=base` to run a single rule layer; the default runs both.

3. **Revise.** Fix slop-layer hits first, applying the canonical fix for
   each high-signal rule:

   | Rule ID                  | Fix                                                                                    |
   | ------------------------ | -------------------------------------------------------------------------------------- |
   | `elevated-register`      | Replace `utilize` with `use`, `commence` with `start`, `facilitate` with `help`, `demonstrate` with `show`. |
   | `filler-adverbs`         | Delete sentence-opening `importantly`, `essentially`, `fundamentally`, `ultimately`; the sentence must earn its own emphasis. |
   | `hedge-stack`            | Keep at most one hedge per sentence; commit to the claim.                              |
   | `em-dash-pivot`          | Replace the em-dash with the mark it stands in for, usually a period or a comma.       |
   | `negation-pivot`         | Rewrite `not X, but Y` as one direct positive claim.                                   |
   | `metaphor-crutch`        | Cut clichés like `north star`, `game changer`, `deep dive`; say the thing plainly.     |
   | `important-to-note`      | Delete the phrase and let the point speak for itself.                                  |
   | `throat-clearing`        | Delete the preamble paragraph entirely and open on the substance.                      |
   | `sycophantic-frame`      | Delete the compliment and answer the question directly instead.                        |

   Then fix base-layer hits, without introducing new slop hits:

   | Rule ID            | Fix                                                                          |
   | ------------------ | ---------------------------------------------------------------------------- |
   | `long-sentence`    | Split past 40 words, at a joint the sentence already has.                    |
   | `long-paragraph`   | Break the paragraph at the point where its second idea starts.               |
   | `passive-voice`    | Put the actor in front of the verb, as in `the analyzer generates the report`. |
   | `padded-verb`      | Cut `allows you to` and `is able to` down to the bare verb.                  |
   | `nominalization`   | Use the verb the noun buries, so `make a decision` becomes `decide`.         |
   | `missing-hyphen`   | Hyphenate the compound that modifies a noun, as in `a command-line tool`.    |
   | `expletive-opener` | Promote the real subject, so `there are two ways` becomes `you can run it two ways`. |

   Each violation's `matchedText` tells you exactly what to change. On
   LLM-backed rules, `suggestedChange` may propose a replacement; use it
   when present. For client-side rules, apply the canonical fix from the
   tables above.

4. **Loop.** Re-run `slop-cop check -` on the revised draft, slop layer
   before base layer each pass, and never trade a base-layer fix for a new
   slop violation. The two layers leave a wide corridor: `staccato-burst`
   fires on a run of three or more sentences of 8 words or fewer, and
   `long-sentence` fires past 40 words, so sentences of 9 to 40 words are
   safe ground. Treat the band as a floor and a ceiling, never as a target
   length. Cap the loop at 4 passes; when a pass fails to reduce the total
   violation count, stop and deliver the previous text. Stop earlier once
   `counts_by_rule` is empty or the only remaining hits are intentional
   stylistic choices you can justify.

   The advisory `readability` object is a Flesch estimate to track across
   revisions; it has no threshold and is never a gate.

5. **Deliver.** Return the revised prose to the user. Do not paste the JSON
   report unless the user explicitly asks for it. Do not announce the loop;
   the point is that the result reads clean, not that the process happened.

## Worked example

Draft the agent wrote, fenced here because it is sample input for the tool
and never prose this document asserts:

```text
In an era of rapid change, it is important to note that, ultimately, the
tapestry of modern software — and this is a paradigm shift — demands
robust collaboration.
```

`slop-cop check -` flags `era-opener`, `important-to-note`,
`filler-adverbs`, three `overused-intensifiers` hits, `metaphor-crutch`,
and two `em-dash-pivot` hits.

Revision:

> Teams build modern software, and version control keeps those teams sane.

A second pass returns an empty `counts_by_rule`, and that revision is what
the user sees.

## Semantic tiers

Two LLM passes layer on top of the 42 client-side rules, selected via
`--llm-effort=off|low|high|auto`. `auto` is the default and resolves to
`high` when the `claude` CLI is on `$PATH`, so usually you don't need to
think about it.

| Effort | Passes run                                 | Extra rules caught                               |
| ------ | ------------------------------------------ | ------------------------------------------------ |
| `off`  | none                                       | none                                             |
| `low`  | sentence tier (Claude Haiku)               | `balanced-take`, `unnecessary-elaboration`, `grandiose-stakes`, `empathy-performance`, `sycophantic-frame`, `throat-clearing`, `pivot-paragraph`, `historical-analogy`, `false-vulnerability`, `triple-construction`, and the base layer's `agentless-passive` |
| `high` | sentence + document (Haiku + Sonnet)       | the low list plus `dead-metaphor`, `one-point-dilution`, `fractal-summaries`, and the base layer's `term-consistency` |

The sugar aliases are `--llm` for `--llm-effort=low` and `--llm-deep` for
`--llm-effort=high`.

Both tiers drive the `claude` CLI, so the check needs no API key. When the
CLI is missing, or a call fails for any reason such as a missing login, a
rate limit, or a timeout, slop-cop skips the auto-enabled pass and reports
its error; the client-side output always returns. Inspect
`llm_effort` and `llm.sentence` / `llm.document` in the JSON report to see
what actually ran.
