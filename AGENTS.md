# AGENTS.md

## Learned User Preferences

- Prefer automated per-push releases driven by GitHub Actions' built-ins over hand-bumped semver tags. Use `github.run_number` as the build number and let GitHub's `/releases/latest` endpoint track "current" — never hand-maintain a mutable `latest` tag.
- Prefer small in-tree subprocess wrappers (`os/exec` + `encoding/json`) over adopting SDKs whose surface area exceeds the need. `severity1/claude-agent-sdk-go` was rejected specifically because it doesn't expose `claude -p --output-format json --json-schema`.
- End-to-end plugin tests belong in a local script (`scripts/test-plugin.sh`), not in CI.
- When GitHub Actions emit Node runtime deprecation warnings, bump the action versions; do not ignore or suppress with env vars.
- When documenting or shipping tools for agent consumption, emit structured JSON on stdout, diagnostics on stderr; no TUIs, prompts, or colored output.

## Learned Workspace Facts

- `main` is the release. Every push to `main` triggers `.github/workflows/release.yml`, which cross-compiles linux/darwin/windows/freebsd × amd64/arm64 and publishes an immutable `v0.1.${{ github.run_number }}` release (tag carries the `v` prefix so Go's module proxy treats it as semver; the binary's printed version string is `0.1.<n>` without the `v`, embedded via `-ldflags -X main.version=0.1.<n>`). Releases are marked `make_latest: true`.
- Binary URLs use GitHub's native redirect: `https://github.com/yasyf/slop-cop/releases/latest/download/slop-cop_<os>_<arch>.tar.gz`. The plugin's first-run bootstrap (`scripts/install-binary.sh` / `.ps1`) fetches from this redirect — GitHub resolves it to the newest release automatically. Never require users to install Go.
- Go toolchain is pinned to 1.26.2 via `.tool-versions` (asdf). Run `asdf install` in the repo root.
- LLM calls shell out to `claude -p --output-format json --json-schema ...` via `internal/llm`. slop-cop never holds an Anthropic API key; it rides the user's `claude` CLI subscription.
- Violation `startIndex` / `endIndex` are UTF-8 byte offsets, not UTF-16 code units as in the upstream JS. Preserve this invariant in detectors and consumers.
- `detectNegationPivot` uses a hand-rolled two-sentence backreference because Go's RE2 `regexp` has no `\2`. Do not "fix" it by introducing backreferences.
- `internal/detectors/word_patterns_test.go` mirrors the upstream Vitest suite (201 subtests). Keep this parity when changing detector behavior.
- The rule catalogue, detectors, word lists, and LLM prompts derive from `awnist/slop-cop`, which was unlicensed at fork time. The Go source is MIT; derived content provenance and compliance guidance live in `NOTICE`. Consult it before broadening use.
- LLM tiers: client-side (35 rules, no network, runs unconditionally on `check`); `--llm` sentence-tier = Claude Haiku (10 rules, chunked at paragraph boundaries around 4000 chars); `--llm-deep` document-tier = Claude Sonnet (3 rules). Both `--llm` / `--llm-deep` auto-enable when `$CLAUDE_PLUGIN_ROOT` or `$CURSOR_PLUGIN_ROOT` is set AND `claude` is on `$PATH`. Auto-enabled passes fail *open* (the failure is reported under `llm.<tier>.error` and the client-side report is still returned); explicit `--llm=true` still hard-fails with exit code 3.
- Plugin manifests live at `.claude-plugin/` (Claude Code) and `.cursor-plugin/` (Cursor); keep both in sync when editing skill or plugin metadata. The active skill is `skills/slop-cop-prose/SKILL.md`.
- Plugin verification runs locally via `scripts/test-plugin.sh`, which spawns `claude -p --plugin-dir` against the checkout. It is intentionally not part of CI.
- `slop-cop check` has a markdown-aware mode that auto-activates on `.md` / `.markdown` / `.mdx` paths (flag: `--markdown=auto|on|off`). It masks non-prose regions (code, URLs, HTML, YAML front matter) via goldmark v1.8+ AST positions and post-filters a couple of structural false positives. The `slop-cop-prose` skill passes `--markdown` unconditionally because LLM drafts are typically markdown-shaped; it's a no-op on plain text.
