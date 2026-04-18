# slop-cop

A Go CLI that detects the rhetorical and structural tells of LLM-generated
prose and emits a structured JSON report. Designed for automated agent
consumption — not humans. No TUI, no highlighting, no interactive prompts.

This is a port of the detection core of
[awnist/slop-cop](https://github.com/awnist/slop-cop), which is a
browser-based editor by [@awnist](https://github.com/awnist). All of the
pattern taxonomy and detector logic comes from that project; this port
re-implements the 35 client-side detectors in Go and shells out to the
[`claude`](https://www.anthropic.com/product) CLI for the optional
semantic and document-level passes.

## Install

```bash
go install github.com/yasyf/slop-cop/cmd/slop-cop@latest
```

Or build from source:

```bash
git clone https://github.com/yasyf/slop-cop
cd slop-cop
go build -o slop-cop ./cmd/slop-cop
```

Requires Go 1.26+ (the repo pins `1.26.2` via `.tool-versions` for asdf).
The optional `--llm` / `--llm-deep` and `rewrite` modes
additionally require the [`claude`](https://docs.claude.com/en/docs/claude-code/overview)
CLI on `$PATH`; slop-cop never needs an Anthropic API key of its own because
`claude -p` uses your Claude subscription.

## Usage

```
slop-cop [command]

Commands:
  check [path|-]      Run detectors; emit JSON report.
  rewrite [path|-]    Rewrite a paragraph via `claude -p`.
  rules               Print the full rule catalogue as JSON.
  version             Print build metadata as JSON.

Exit codes:
  0  success (including "no violations found")
  2  input/IO error
  3  claude subprocess error
  4  flag/usage error
```

Input is the positional argument (`-` or omitted for stdin). Output is JSON
on stdout; diagnostics go to stderr.

### `check`

```bash
# File input
slop-cop check article.md

# Stdin
cat article.md | slop-cop check

# With Claude-backed semantic + document passes
slop-cop check article.md --llm --llm-deep --pretty
```

Example output (trimmed):

```json
{
  "text_length": 135,
  "violations": [
    {
      "ruleId": "era-opener",
      "startIndex": 0,
      "endIndex": 12,
      "matchedText": "In an era of"
    },
    {
      "ruleId": "important-to-note",
      "startIndex": 27,
      "endIndex": 50,
      "matchedText": "it is important to note"
    }
  ],
  "counts_by_rule": { "era-opener": 1, "important-to-note": 1 },
  "counts_by_category": { "rhetorical": 1, "word-choice": 1 }
}
```

Flags:

| Flag                   | Default                       | Purpose                                         |
| ---------------------- | ----------------------------- | ----------------------------------------------- |
| `--llm`                | off                           | Sentence-tier semantic pass (Claude Haiku)      |
| `--llm-deep`           | off                           | Document-tier structural pass (Claude Sonnet)   |
| `--claude-bin`         | `claude`                      | Path to the `claude` CLI                        |
| `--sentence-model`     | `claude-haiku-4-5-20251001`   | Model slug for `--llm`                          |
| `--document-model`     | `claude-sonnet-4-6`           | Model slug for `--llm-deep`                     |
| `--sentence-timeout`   | `30s`                         | Timeout per sentence-pass chunk                 |
| `--document-timeout`   | `60s`                         | Timeout for the document pass                   |
| `--pretty`             | off                           | Indent JSON output                              |

### `rewrite`

Runs the `rewriteParagraph` flow from the original source: builds a system
prompt from the default rewrite directives (plus any rule IDs you pass) and
asks Claude to rewrite the input.

```bash
slop-cop rewrite draft.txt --rules filler-adverbs,hedge-stack --pretty
```

Output:

```json
{
  "rewritten": "…",
  "applied_rules": ["filler-adverbs", "hedge-stack"]
}
```

### `rules`

Dumps rule metadata so agents don't have to hard-code it.

```bash
slop-cop rules --llm-only --pretty
slop-cop rules --category word-choice
```

## Violation shape

```ts
type Violation = {
  ruleId: string            // stable rule identifier (see `slop-cop rules`)
  startIndex: number        // byte offset into the UTF-8 input, inclusive
  endIndex: number          // byte offset, exclusive
  matchedText: string       // text[startIndex:endIndex]
  explanation?: string      // optional extra context (LLM passes, and a few
                            //   client-side detectors like hedge-stack set this)
  suggestedChange?: string  // LLM-supplied replacement, "" to suggest deletion
}
```

**Byte offsets.** Offsets are bytes into the UTF-8 input, not UTF-16 code
units as in the original JavaScript. Go's `regexp` and `strings` packages
work in bytes by default, so consumers doing `text[v.StartIndex:v.EndIndex]`
in Go (or any language that indexes strings by bytes) will get the right
substring. If you're slicing the original string in a language that indexes
by code units (JavaScript, Java), account for the difference around
non-ASCII characters.

## Detection tiers

**Client-side (instant).** 35 regex + structural detectors. No external
calls. These run unconditionally on `check`.

**`--llm` (sentence-tier).** Shells out to `claude -p --output-format json
--json-schema ...` with Claude Haiku. Large inputs are chunked at paragraph
boundaries (4000-char threshold, 3500-char target) and analysed in parallel,
then merged and deduplicated by `ruleId + matchedText`. 10 rules:
`triple-construction`, `throat-clearing`, `sycophantic-frame`,
`balanced-take`, `unnecessary-elaboration`, `empathy-performance`,
`pivot-paragraph`, `grandiose-stakes`, `historical-analogy`,
`false-vulnerability`.

**`--llm-deep` (document-tier).** Shells out to Claude Sonnet. 3 rules:
`dead-metaphor`, `one-point-dilution`, `fractal-summaries`.

## Rules

48 total: 35 client-side, 10 sentence-tier (LLM), 3 document-tier (LLM). See
the [upstream README](https://github.com/awnist/slop-cop#patterns-detected)
for the full pattern list, or run `slop-cop rules --pretty` locally.

## Differences from the upstream

- Go CLI instead of a browser UI; no editor, no URL hash sync, no
  contenteditable.
- LLM calls go through the `claude` CLI (subscription auth) rather than
  direct Anthropic API requests.
- Offsets are UTF-8 byte indices rather than JavaScript UTF-16 units.
- `detectNegationPivot` reimplements the two-sentence backreference
  case by hand because Go's `regexp` (RE2) has no `\2`.

All other detection logic is a 1:1 port; the 196 subtests in
[`internal/detectors/word_patterns_test.go`](internal/detectors/word_patterns_test.go)
mirror the original Vitest suite.

## Development

The repo pins Go 1.26.2 via `.tool-versions`. With
[asdf](https://asdf-vm.com) installed, run `asdf install` in the repo root
to pick it up; otherwise install Go 1.26 manually.

```bash
go test ./...
go vet ./...
go build ./...
```

## License & credit

The Go source in this repository (the CLI, subprocess plumbing, build
system, documentation, tests) is released under the [MIT License](LICENSE).

The pattern taxonomy, rule catalogue, detector algorithms, word lists, and
LLM prompts are derived from [awnist/slop-cop](https://github.com/awnist/slop-cop)
by [@awnist](https://github.com/awnist). At the time this port was made,
that upstream repository carried no open-source licence; see
[NOTICE](NOTICE) for the full provenance, attribution, and compliance
guidance. If you plan to use this tool beyond personal use or contributions
back to the upstream author, please reach out to @awnist to clarify
licensing of the derived content.

Original source rules: [LLM_PROSE_TELLS.md](https://git.eeqj.de/sneak/prompts/src/branch/main/prompts/LLM_PROSE_TELLS.md) (MIT, © sneak),
[Wikipedia: Signs of AI Writing](https://en.wikipedia.org/wiki/Wikipedia:Signs_of_AI_writing) (CC BY-SA 4.0),
[tropes.md](https://tropes.fyi/tropes-md).
