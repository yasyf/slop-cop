---
name: slop-cop-prose
description: Detects and fixes LLM-generated prose tells and plain-language clarity problems using the slop-cop CLI. Triggers whenever the user asks the agent to write, draft, revise, polish, or edit natural-language prose (blog posts, docs, PR descriptions, commit messages, release notes, marketing copy, emails). The agent pipes its own draft through `slop-cop check -`, revises against the violation report, and delivers only the cleaned result.
allowed-tools: Bash(slop-cop:*), Bash(bash:*), Read
---

# Slop Cop (prose)

Before returning a piece of prose to the user, whether a blog paragraph, a
doc, a PR description, a commit message, a release note, or an email, run it
through `slop-cop` and revise. The check covers 226 rules in three layers. The
google layer is the house style, 169 rules ported from the Google developer
documentation style guide, covering voice, tense, headings, links,
punctuation, and word choice. The slop layer's 48 rules catch LLM writing
tells such as overused intensifiers, filler adverbs, negation pivots, em-dash
abuse, throat-clearing, and hedge stacks. The base layer's nine rules catch
plain-language clarity problems such as 40-word sentences, padded verbs, and
passives that demote their actor, and those fire on human-sounding prose just
as readily.

Where the google layer and an older rule disagree, the google layer wins the
span it matched, and the older rule keeps firing everywhere else. You do not
have to arbitrate that; slop-cop already has.

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

## Resolve the binary

Resolve once per session, in this order, and verify the result before you
trust it.

1. `${CLAUDE_PLUGIN_ROOT}/bin/slop-cop`, or `${CURSOR_PLUGIN_ROOT}/bin/slop-cop`
   in Cursor. That committed wrapper execs the release pinned in
   `bin/slop-cop.binrun` through binrun, downloading and caching it on first
   call, so it always runs the version this skill documents.
2. `slop-cop` on `$PATH`, for CI, scripting, or a standalone install.

When both variables are unset, the plugin root is the directory two levels
above this file.

```bash
PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-${CURSOR_PLUGIN_ROOT:-}}"
if [ -x "${PLUGIN_ROOT}/bin/slop-cop" ]; then
  SLOP_COP="${PLUGIN_ROOT}/bin/slop-cop"
else
  SLOP_COP=slop-cop
fi
"$SLOP_COP" --version
```

`--version` prints one bare version line. A binary that answers
`unknown flag: --version` and exits 2 predates this skill: it has no google
layer, no `--strict`, no `--format=compact`, and none of the report fields
described below. Discard it and resolve again. The first report you read
must carry `ran`, `version`, and `binary_path`.

- Never take a binary from a plugin data directory such as
  `~/.claude/plugins/data/<plugin>/bin`. Data directories persist across
  plugin versions and hold downloads from older packaging schemes; the plugin
  root the wrapper lives under is a `plugins/cache` directory. When
  `binary_path` lands under `plugins/data`, discard it and resolve again.
- Never hardcode a discovered absolute path into a later command or a memory
  note; that includes `binary_path`. Re-resolve from the plugin root every
  session.
- A rule ID you do not recognize, or a flag rejected as unknown, is a version
  mismatch, not a broken tool. Compare the report's `version` with
  `version.static` in `${PLUGIN_ROOT}/bin/slop-cop.binrun`.

## Read a report

Default to `--format=compact`. It prints one line per violation, then a
`counts` trailer:

```text
ruleId<TAB>line:col<TAB>matchedText
counts<TAB>ruleId=n ruleId=n
```

Line and column are 1-based, and a clean run prints the trailer alone. Use
the JSON report, `--format=json` or no flag, when you need byte offsets,
`suggestedChange`, the `llm` outcome, or the `rules` sidecar.

A killed or crashed run leaves empty stdout, and without `--strict` its exit
code matches a clean pass. The payload is the only marker of a completed run,
so test for the marker, never for a byte count or an exit code.

| Format    | Marker of a completed run                |
| --------- | ---------------------------------------- |
| JSON      | The `ran` field is present.              |
| `compact` | The `counts` trailer line is present.    |

Empty stdout, or `exit 137`, means an unchecked draft, never a clean one.
Run the check again.

JSON fields that matter in the loop:

| Field                | Meaning                                                                                        |
| -------------------- | ---------------------------------------------------------------------------------------------- |
| `ran`                | Always `true`; its presence is the marker of a completed run.                                  |
| `version`, `binary_path` | The build that produced the report. Read them when a rule ID or flag surprises you.        |
| `violations`         | One entry per hit, with `ruleId`, `startIndex` and `endIndex` as UTF-8 byte offsets, `matchedText`, and on LLM-backed rules `explanation` and `suggestedChange`. |
| `counts_by_rule`     | Empty when the draft is clean.                                                                 |
| `rules`              | Fix guidance once per fired rule ID, as `name`, `category`, `tip`, and `rewriteHint`. Absent when nothing fired. |
| `llm_effort`, `llm`  | What the semantic tiers resolved to and whether each pass ran; `llm.<tier>.error` names a failure. |
| `config`             | The `.slopcop.toml` that filtered the run, when one applied.                                   |
| `readability`        | An advisory Flesch estimate over at least 100 words of prose. Track it across revisions; it gates nothing. |

The `rules` sidecar is the general answer to how a rule gets fixed. The
tables under [Revise](#loop) cover the rules that fire most; for any other
ID, follow the sidecar's `tip`.

Exit codes:

| Code | Meaning                                                              |
| ---- | -------------------------------------------------------------------- |
| 0    | Completed, clean or with violations. `--strict` is opt-in.           |
| 1    | Completed with violations under `--strict`.                          |
| 2    | IO error, including a `--config` path that does not exist.           |
| 3    | An explicit LLM pass (`--llm`, `--llm-deep`, `--llm-effort=low\|high`) failed. Auto-enabled passes fail open instead. |
| 4    | A bad flag, `--lang`, `--standard`, `$SLOP_COP_LLM` value, config key, or rule ID. |

## Latency

| Work                                       | Cost                                   |
| ------------------------------------------ | -------------------------------------- |
| Client-side rules (`--no-llm`)             | About 0.3s.                            |
| Sentence tier, the default with `codex` on `$PATH` | Adds about 10s. That is the provider's floor; slop-cop adds nothing measurable to it. |
| Document tier (`--llm-deep`)               | Adds about 7s more.                    |

`--llm-effort`, its aliases, and `$SLOP_COP_LLM` are the only variables that
move latency. Reading from stdin against a path, `--lang`, `--format`,
`--lines`, and the size of a normal draft make no measurable difference. A
remembered "stdin is slow" or "slop-cop hangs" comes from a run that carried
the sentence tier. Before acting on any such memory, probe:

```bash
printf 'Probe.\n' | "$SLOP_COP" check --no-llm --format=compact -
```

That answers in under a second. If it does not, the binary or the machine is
the problem, never the input form.

## Loop

1. **Draft.** Write the prose the user asked for.

2. **Check, client-side only.** Iterate with `--no-llm`; the semantic tier
   runs once, in step 5.

   ```bash
   printf '%s' "$DRAFT" | "$SLOP_COP" check --lang=markdown --no-llm --format=compact -
   ```

   For a file, pass its path and let the extension pick the mode:

   ```bash
   "$SLOP_COP" check docs/guide.md --no-llm --format=compact
   ```

   For an edited section, or a PR body you patched in place, add `--lines`.
   Detectors still scan the whole document for context; the report keeps
   only violations that begin in the range. Post-filtering the JSON with
   `jq` does the same job slower:

   ```bash
   "$SLOP_COP" check docs/guide.md --lines 40:80 --no-llm --format=compact
   ```

   Pass `--lang=markdown` for prose drafts on stdin. LLM drafts are
   typically markdown-shaped, full of code fences, inline code, links, and
   headings, and markdown mode masks those non-prose regions so detectors
   only see the actual writing. It is safe on plain prose. On a path,
   markdown extensions get markdown mode, `.html` and `.htm` get html mode,
   and `.jsx`, `.tsx`, `.ts`, and `.js` get the matching tree-sitter mode,
   which masks code so detectors only see comments, string literals,
   template quasis, and JSX text. Pass `--standard=slop`, `--standard=base`,
   or `--standard=google` to run a single rule layer; the default runs all
   three.

3. **Revise.** Fix google-layer hits first, since that layer defines the
   house style. These are the rules that fire most on ordinary prose:

   | Rule ID                    | Fix                                                                                      |
   | -------------------------- | ---------------------------------------------------------------------------------------- |
   | `reader-address-person`    | Address the reader as `you`; keep `the user` for the reader's own end users, and drop `we` and `let's` from procedures. |
   | `passive-by-agent`         | Put the actor in front of the verb, so `X is validated by the server` becomes `The server validates X`. |
   | `impersonal-recommendation`| Give the advice an owner with `We recommend that you ...`; use `must` for a requirement and `can` for an option. |
   | `trivializing-difficulty`  | Delete `simply`, `easy`, `just`, `merely`, `quickly`; if effort matters, give a number instead. |
   | `future-tense-behavior`    | Write the present tense: `the server returns a 200`, never `will return`.                |
   | `time-bound-qualifier`     | Delete `currently` and `at present`; name the version or the date when the claim is bound to one. |
   | `superlative-product-claim`| Replace the superlative with a scoped, sourced comparison, or describe the behavior instead. |
   | `heading-sentence-case`    | Capitalize the first word and proper nouns only, so `Getting Started With The API` becomes `Get started with the API`. |
   | `vague-link-text`          | Replace `click here` and `this document` with the destination's title.                   |
   | `plain-word-swap`          | Use the everyday word: `use`, `start`, `run`, `so`, `before`, `help`.                    |
   | `multiword-for-single-word`| Collapse the stock phrase, so `has the ability to` becomes `can` and `a number of` becomes `some`. |
   | `latin-abbreviation`       | Write `that is` for `i.e.` and `for example` for `e.g.`, and finish the list instead of trailing off with `etc.` |
   | `serial-comma`             | Put a comma before the `and` or `or` introducing the last item.                          |

   Then fix slop-layer hits, applying the canonical fix for each
   high-signal rule:

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

   Then fix base-layer hits, without introducing new google or slop hits:

   | Rule ID            | Fix                                                                          |
   | ------------------ | ---------------------------------------------------------------------------- |
   | `long-sentence`    | Split past 40 words, at a joint the sentence already has.                    |
   | `long-paragraph`   | Break the paragraph at the point where its second idea starts.               |
   | `passive-voice`    | Put the actor in front of the verb, as in `the analyzer generates the report`. |
   | `padded-verb`      | Cut `allows you to` and `is able to` down to the bare verb.                  |
   | `nominalization`   | Use the verb the noun buries, so `make a decision` becomes `decide`.         |
   | `missing-hyphen`   | Hyphenate the compound that modifies a noun, as in `a command-line tool`.    |
   | `expletive-opener` | Promote the real subject, so `there are two ways` becomes `you can run it two ways`. |

   Each violation's `matchedText` tells you exactly what to change. For a
   rule ID outside these tables, read `rules.<id>.tip` in the JSON report.
   On LLM-backed rules, `suggestedChange` may propose a replacement; use it
   when present.

4. **Loop.** Re-run the `--no-llm` check on the revised draft, google layer
   before slop layer before base layer each pass, and never trade a fix in
   one layer for a new violation in another. The layers leave a wide
   corridor: `staccato-burst` fires on a run of three or more sentences of 8
   words or fewer, and `long-sentence` fires past 40 words, so sentences of 9
   to 40 words are safe ground. Treat the band as a floor and a ceiling,
   never as a target length. Cap the loop at four passes; when a pass fails to
   reduce the total violation count, stop and deliver the previous text. Stop
   earlier once the `counts` trailer is bare or the only remaining hits are
   intentional stylistic choices you can justify.

5. **Semantic pass, once.** Run the final draft without `--no-llm`, in JSON
   so you can read the outcome:

   ```bash
   printf '%s' "$DRAFT" | "$SLOP_COP" check --lang=markdown -
   ```

   With `codex` on `$PATH` the default `--llm-effort=auto` resolves to `low`
   and runs the sentence tier; `llm.sentence.ran` confirms it, and
   `llm.sentence.error` explains a skip. Fix its hits, then confirm with one
   more `--no-llm` compact check that the fixes introduced no client-side
   hits. Running the semantic tier inside the revision loop costs about 10s
   per pass for findings that rarely change between drafts; running it once
   here costs 10s per document.

6. **Deliver.** Return the revised prose to the user. Do not paste the
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

The `--no-llm` check flags `era-opener`, `important-to-note`,
`filler-adverbs`, three `overused-intensifiers` hits, `metaphor-crutch`,
and two `em-dash-pivot` hits.

Revision:

> Teams build modern software, and version control keeps those teams sane.

A second pass returns a bare `counts` trailer, the semantic pass adds
nothing, and that revision is what the user sees.

## Semantic tiers

Two LLM passes layer on top of the 178 client-side rules, selected with
`--llm-effort=off|low|high|auto`. `auto` is the default and resolves to
`low` when the `codex` CLI is on `$PATH`, else `off`.

| Effort | Passes run          | Rules added                                                                                                      |
| ------ | ------------------- | ---------------------------------------------------------------------------------------------------------------- |
| `off`  | none                | none                                                                                                             |
| `low`  | sentence tier       | 40 rules. The slop layer's `balanced-take`, `unnecessary-elaboration`, `grandiose-stakes`, `empathy-performance`, `sycophantic-frame`, `throat-clearing`, `pivot-paragraph`, `historical-analogy`, `false-vulnerability`, and `triple-construction`; the base layer's `agentless-passive`; and 29 google rules covering comma placement, ambiguous pronouns and demonstratives, noun stacks, jargon, and register. |
| `high` | sentence + document | The `low` set plus 8 rules. `dead-metaphor`, `one-point-dilution`, `fractal-summaries`, the base layer's `term-consistency`, and the google layer's `repeated-procedure`, `unrecommended-option-menu`, `audience-consistency`, and `undefined-jargon`. |

`slop-cop rules --llm-only` lists every LLM rule with its tier.

| Control                  | Effect                                                   |
| ------------------------ | -------------------------------------------------------- |
| `--llm-effort=<level>`   | The underlying control; wins over everything below.      |
| `--no-llm`               | `--llm-effort=off`. Wins over `--llm` and `--llm-deep`.  |
| `--llm-deep`             | `--llm-effort=high`.                                     |
| `--llm`                  | `--llm-effort=low`.                                      |
| `$SLOP_COP_LLM`          | The same four values; applies when no flag does. A bad value exits 4. |

Both detection tiers run `gpt-5.6-luna:low` through the `codex` CLI, with a
45s budget per sentence-pass chunk and 90s for the document pass. `rewrite`
and `plainify` run the `claude` CLI. Neither needs an API key. An
auto-enabled pass that fails, whether codex is missing, logged out, rate
limited, or timing out, is skipped. The report carries the client-side
results, `llm.<tier>.ran` is `false`, and `llm.<tier>.error` names the
cause. An explicitly requested pass that fails exits 3.

## Suppress a rule

A rule that keeps firing on a construction the house style mandates, such as
`colon-elaboration` on a required `Context:` label, belongs in a
`.slopcop.toml`, not in a fresh argument every session. slop-cop walks up from
the input path to find one, `--config` overrides discovery, and the report's
`config` field names the file that applied. The schema, with `disable`,
`enable_only`, and per-glob `[[overrides]]`, is in the README under
[Suppress rules with .slopcop.toml](../../README.md#suppress-rules-with-slopcoptoml).
A suppressed rule never reaches the semantic tiers' prompt either. An unknown
rule ID in the file exits 4 naming it.

## Plain-English twins

`slop-cop check` grades one piece of prose. `slop-cop plainify` writes a
second one, a plain-English twin of text that is precise but hard to read
cold, such as a register entry, a review finding, or a recorded decision. The
twin sits beside the original, and the original stays authoritative.

Reach for it whenever the reader sits outside the project. A finding that
names `DQ4` reads fine to whoever wrote it and not at all to anyone else, and
the same goes for claudish, the prose an agent writes for other agents.
Plainify turns both into something a reader follows on one pass.

```bash
"$SLOP_COP" plainify draft.md --max-words 120 --name-by-title --glossary titles.json
```

The rewrite keeps every fact, name, number, and file path, and leaves fenced
and inline code alone. `--max-words` and `--forbid <regex>` reach the model as
instructions and are graded again once it answers, so
`--forbid '\b(DQ|A|Q|V)\d+\b'` catches an identifier the rewrite kept. What a
retry still misses lands in the result's `truncated` and `violations` fields
instead of being dropped. `--json` reads an array of `{"id","text"}` entries
and returns one result per entry, in the order they arrived. The
`/slop-cop-plainify` command runs the same rewrite as a one-off.
