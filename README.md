# ![slop-cop](docs/assets/readme-banner.webp)

**Never ship `delve` again.** slop-cop reads your agent's prose with 57 rules and returns a JSON rap sheet the agent revises against, in CI, pre-commit, or inline.

[![CI](https://github.com/yasyf/slop-cop/actions/workflows/ci.yml/badge.svg)](https://github.com/yasyf/slop-cop/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/yasyf/slop-cop)](https://github.com/yasyf/slop-cop/releases/latest)
[![MIT license](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

## Get started

```bash
brew install yasyf/tap/slop-cop
slop-cop check draft.md --pretty
```

<img src="docs/assets/demo.png" alt="Terminal running 'slop-cop check draft.md --pretty' — a JSON report flagging era-opener, 'delve', and a negation pivot" width="700">

Driving with an agent? Paste this:

```
/plugin marketplace add yasyf/slop-cop
/plugin install slop-cop@slop-cop
```

<details>
<summary>No plugin support? Paste this instead.</summary>

```text
Install slop-cop with `brew install yasyf/tap/slop-cop`. Then run
`slop-cop check README.md --pretty` and revise the file until the
violations array is empty.
```

</details>

---

## Use cases

### Make your agent self-edit before you ever see the draft

You ask for a PR description and get "In today's fast-paced world" back, and then you spend the review playing copy editor. Install the plugin and its skill runs this loop on every draft, before the agent replies:

```bash
printf '%s' "$draft" | slop-cop check --lang=markdown -
```

The agent walks the `violations` array, rewrites each `matchedText`, and re-checks until `counts_by_rule` is `{}`. You see the clean revision, never the slop.

### Catch the sentence nobody can parse, whoever typed it

The slop layer polices prose that sounds like an LLM, and the new base layer underneath polices prose that is merely unclear. Its 9 plain-language rules flag the 45-word sentence, the padded `allows you to`, the passive that demotes its actor, and the paragraph carrying three ideas, in human prose and agent prose alike:

```bash
slop-cop check docs/spec.md --standard=base
```

`--standard` picks the layer set. `all` is the default and runs both, `slop` runs the slop layer alone, and `base` runs the base layer alone; an unknown value exits `4`.

### Block era-openers and hedge stacks in CI and pre-commit

Slop merges because nobody wants to be the reviewer who flags tone. Make the machine do it:

```bash
slop-cop check docs/announcement.md | jq -e '.violations == []'
```

`check` always exits 0 and puts the verdict in the JSON; `jq -e` turns the empty `violations` array into the pass/fail bit. The 42 client-side rules make no network calls, so the gate costs milliseconds.

If you gated CI on an earlier slop-only release, this gate now fails on base-layer hits too. Pass `--standard=slop` to reproduce the old gate; its report is byte-identical to what earlier releases printed.

### Lint only the prose hiding in JSDoc, JSX, and string literals

Your linter has opinions about semicolons and none about the `seamless synergy` in your hero copy:

```bash
slop-cop check src/Hero.tsx
```

tree-sitter masks every non-prose byte before detectors run, so hits land only on comments, string literals, and JSX text. An `In an era of` JSDoc opener and a `negation-pivot` inside a string literal both get flagged, with offsets that index the original file.

## Commands

| Command | What it does |
| --- | --- |
| `check [path\|-]` | Run detectors; emit the JSON report. |
| `rewrite [path\|-]` | Rewrite a paragraph via the `claude` CLI, optionally targeting `--rules`. |
| `rules` | Print the rule catalogue as JSON; filter with `--category` or `--llm-only`. |
| `version` | Print build metadata as JSON. |

Input is the positional argument; pass `-` or omit it to read stdin. Run `slop-cop check --help` for the full flag list, or `slop-cop rules --pretty` for the taxonomy of all 57 rules. Exit codes are `0` for success, `2` for an input or IO error, `3` for a claude subprocess failure, and `4` for a usage error.

## How it works

The interface assumes an agent is driving, so slop-cop prints JSON on stdout and diagnostics on stderr, with no TUI, no highlighting, and no prompts. `check` runs the 42 instant client-side rules, regex and structural, then up to two Claude-backed tiers. The sentence tier adds 11 rules on Claude Haiku under `--llm`, the document tier adds 4 more on Claude Sonnet under `--llm-deep`, and the full catalogue is 57 rules. `--llm-effort` is the underlying control, taking `off`, `low`, `high`, or `auto`. The default `auto` resolves to `low` whenever the `claude` CLI is on `$PATH`, so the sentence tier runs and the slower document tier waits for an explicit `--llm-deep`. An auto-enabled pass that fails reports the error in the report's `llm` field while the client-side results still return. The LLM tiers and `rewrite` drive the `claude` CLI, so slop-cop never needs an Anthropic API key; it rides your Claude subscription.

When the base layer runs over at least 100 words of prose, the report also carries an advisory `readability` object with estimated Flesch reading-ease and grade-level scores. It never produces a violation and never moves the exit code; it is a number to track across revisions, and it disappears under `--standard=slop`.

`--lang` parses the input and masks non-prose bytes before detectors run; `auto` picks by file extension, and the explicit modes are `text`, `markdown`, `html`, `jsx`, `tsx`, `ts`, and `js`. Masking preserves length and newline offsets, so violation offsets index the original input. Add `--lines 50:80` to report only violations beginning in an edited range while still scanning the whole document for context.

> [!WARNING]
> `startIndex` and `endIndex` are UTF-8 byte offsets. Slicing by UTF-16 code units in JavaScript or Java corrupts the spans; convert first.

## The agent plugin

The [`slop-cop-prose`](skills/slop-cop-prose/SKILL.md) skill triggers whenever you ask the agent to write, revise, or polish prose, and keeps the loop silent; the agent never announces it. A [`/slop-cop-check`](commands/slop-cop-check.md) command runs a one-off report without rewriting anything.

<details>
<summary>Claude Code</summary>

Install with the two `/plugin` commands under [Get started](#get-started). The plugin ships a committed [`bin/slop-cop`](bin/slop-cop) wrapper that resolves the exact release pinned in [`bin/slop-cop.binrun`](bin/slop-cop.binrun) through [binrun](https://github.com/yasyf/binrun), a checksum-verified download cached after the first call and pre-warmed on session start, with no Go toolchain required. Verify end-to-end with [`scripts/test-plugin.sh`](scripts/test-plugin.sh).

</details>

<details>
<summary>Cursor</summary>

Open the Plugins panel and install from Git URL `https://github.com/yasyf/slop-cop`. The same skill and first-run bootstrap apply, keyed off `$CURSOR_PLUGIN_ROOT`.

</details>

---

The Go source is licensed under [MIT](LICENSE). The slop layer's taxonomy, detectors, word lists, and prompts derive from [awnist/slop-cop](https://github.com/awnist/slop-cop) by [@awnist](https://github.com/awnist), which carried no license at port time; read [NOTICE](NOTICE) before use beyond personal. The base layer's rules are original to this project, inspired by the [ASD-STE100](https://asd-ste100.org) Simplified Technical English standard; NOTICE records the scope of that inspiration. The slop rules trace to [LLM_PROSE_TELLS.md](https://git.eeqj.de/sneak/prompts/src/branch/main/prompts/LLM_PROSE_TELLS.md) under MIT by sneak, the Wikipedia essay [Signs of AI writing](https://en.wikipedia.org/wiki/Wikipedia:Signs_of_AI_writing) under CC BY-SA 4.0, and [tropes.md](https://tropes.fyi/tropes-md).
