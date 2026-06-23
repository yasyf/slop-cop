# slop-cop

A Go CLI that detects the rhetorical and structural tells of LLM-generated prose and emits a structured JSON report.

Built for agents, not humans: no TUI, no highlighting, no prompts — just JSON on stdout, diagnostics on stderr. It ships with a plug-and-play skill so a coding agent runs `slop-cop` on its own drafts and silently revises before replying.

## Install

Most people use slop-cop through their coding agent. The skill activates whenever you ask the agent to write, edit, or polish prose.

**Claude Code:**

```
/plugin marketplace add yasyf/slop-cop
/plugin install slop-cop@slop-cop
```

**Cursor:** open the Plugins panel and install from Git URL `https://github.com/yasyf/slop-cop`.

On the first draft, the skill runs `scripts/install-binary.sh` (or `.ps1` on Windows) to download the prebuilt binary for your platform from the rolling [`latest`](https://github.com/yasyf/slop-cop/releases/tag/latest) release. No Go toolchain required. Verify end-to-end with [`scripts/test-plugin.sh`](scripts/test-plugin.sh).

To invoke `slop-cop` directly from CI, pre-commit hooks, or scripts, install the binary:

```bash
# Homebrew (macOS)
brew install yasyf/tap/slop-cop

# Or from source
go install github.com/yasyf/slop-cop/cmd/slop-cop@latest
```

The `--llm` modes and `rewrite` additionally require the [`claude`](https://docs.claude.com/en/docs/claude-code/overview) CLI on `$PATH`. slop-cop never needs an Anthropic API key — `claude -p` uses your Claude subscription.

## Quickstart

```bash
slop-cop check article.md --pretty
```

```json
{
  "text_length": 135,
  "violations": [
    {
      "ruleId": "era-opener",
      "startIndex": 0,
      "endIndex": 12,
      "matchedText": "In an era of"
    }
  ],
  "counts_by_rule": { "era-opener": 1 },
  "counts_by_category": { "rhetorical": 1 }
}
```

Input is the positional argument (`-` or omitted for stdin). The mode is picked from the file extension; pass `--lang` to force one (`text`, `markdown`, `html`, `jsx`, `tsx`, `ts`, `js`). Add `--llm` for Claude-backed semantic passes and `--lines 50:80` to report only violations beginning in an edited range.

## Commands

| Command | What it does |
| --- | --- |
| `check [path\|-]` | Run detectors; emit a JSON violation report. |
| `rewrite [path\|-]` | Rewrite a paragraph via `claude -p`, optionally targeting `--rules`. |
| `rules` | Print the rule catalogue as JSON (`--category`, `--llm-only`). |
| `version` | Print build metadata as JSON. |

Run `slop-cop check --help` for the full flag list, or `slop-cop rules --pretty` for the rule taxonomy. Exit codes: `0` success, `2` input/IO error, `3` claude subprocess error, `4` usage error.

## How it works

`check` runs 35 instant client-side detectors (regex + structural, no external calls), then optionally Claude-backed passes: a sentence tier (Haiku, 10 rules) under `--llm` and a document tier (Sonnet, 3 rules) under `--llm-deep` — 48 rules total. `--llm-effort` (`off｜low｜high｜auto`) is the underlying control; `--llm`/`--llm-deep` are sugar for `low`/`high`. Under a plugin (`$CLAUDE_PLUGIN_ROOT` or `$CURSOR_PLUGIN_ROOT` set) with `claude` reachable, `auto` resolves to `high`; the resolved effort and per-tier outcomes are reported in the JSON, and an unreachable `claude` degrades gracefully (the failure lands in an `error` field while client-side results still return).

`--lang` parses code and masks every non-prose byte before detectors run, so you get hits on prose only — JSDoc, string literals, JSX text, HTML copy, markdown. Masking preserves length and newline offsets, so byte offsets in violations index the original UTF-8 input (bytes, not UTF-16 code units — account for that when slicing in JavaScript or Java). See [`skills/slop-cop-prose/SKILL.md`](skills/slop-cop-prose/SKILL.md) for the exact agent instructions.

## Development

The repo pins Go 1.26.2 via `.tool-versions` (run `asdf install`, or install Go 1.26 manually).

```bash
go test ./...
go vet ./...
go build ./...
```

`main` is the release: every push triggers [release.yml](.github/workflows/release.yml), which cross-compiles and publishes an immutable `v0.1.<run_number>` GitHub Release marked Latest.

## License & credit

The Go source in this repository is released under the [MIT License](LICENSE).

The pattern taxonomy, rule catalogue, detector algorithms, word lists, and LLM prompts are derived from [awnist/slop-cop](https://github.com/awnist/slop-cop) by [@awnist](https://github.com/awnist), which carried no open-source licence at port time; see [NOTICE](NOTICE) for provenance and compliance guidance. If you plan to use this beyond personal use, reach out to @awnist to clarify licensing of the derived content.

Original source rules: [LLM_PROSE_TELLS.md](https://git.eeqj.de/sneak/prompts/src/branch/main/prompts/LLM_PROSE_TELLS.md) (MIT, © sneak), [Wikipedia: Signs of AI Writing](https://en.wikipedia.org/wiki/Wikipedia:Signs_of_AI_writing) (CC BY-SA 4.0), [tropes.md](https://tropes.fyi/tropes-md).
