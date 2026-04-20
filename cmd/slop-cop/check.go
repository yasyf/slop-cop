package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yasyf/slop-cop/internal/detectors"
	_ "github.com/yasyf/slop-cop/internal/htmllang" // lang registry
	_ "github.com/yasyf/slop-cop/internal/jslang"   // lang registry
	"github.com/yasyf/slop-cop/internal/lang"
	"github.com/yasyf/slop-cop/internal/llm"
	_ "github.com/yasyf/slop-cop/internal/markdown" // lang registry
	"github.com/yasyf/slop-cop/internal/rules"
	"github.com/yasyf/slop-cop/internal/types"
)

// checkReport is the JSON document emitted by `slop-cop check`. Counts are
// denormalised so callers don't have to walk the violations slice.
type checkReport struct {
	TextLength       int                             `json:"text_length"`
	Violations       []types.Violation               `json:"violations"`
	CountsByRule     map[string]int                  `json:"counts_by_rule"`
	CountsByCategory map[types.ViolationCategory]int `json:"counts_by_category"`
	// Lang reports which input-language masker ran. "text" means no masking;
	// other values ("markdown", "html", "jsx", "tsx", "ts", "js") match the
	// registered Analyzer names in internal/lang.
	Lang string `json:"lang"`
	// LLMEffort is the effort level slop-cop resolved to for this run:
	// "off", "low", or "high". Emitted alongside LLM so agents can read
	// the setting even when no pass ran.
	LLMEffort string `json:"llm_effort"`
	// LLM captures per-tier outcomes. Omitted entirely when effort=off.
	LLM *llmReport `json:"llm,omitempty"`
}

// llmReport captures the outcome of the two LLM passes. Either sub-field
// may be nil if that specific pass wasn't attempted.
type llmReport struct {
	Sentence *llmPassStatus `json:"sentence,omitempty"`
	Document *llmPassStatus `json:"document,omitempty"`
}

// llmPassStatus describes a single LLM pass's outcome.
type llmPassStatus struct {
	// Auto is true when the pass was enabled by the auto-default rather
	// than an explicit flag. Drives fail-open vs fail-closed semantics.
	Auto bool `json:"auto"`
	// Ran is true when the pass executed successfully and contributed
	// violations to the report.
	Ran bool `json:"ran"`
	// Error holds the pass's error message when Ran is false. Empty when
	// the pass succeeded or wasn't attempted.
	Error string `json:"error,omitempty"`
}

// resolveLang picks the lang.Analyzer for this invocation based on --lang.
// Returns the selected Analyzer (nil for "text") and the lang name to record
// in checkReport.Lang. "auto" detects from the file extension, falling back
// to "text" for stdin or an unrecognised extension. An unknown explicit
// value is a usage error.
func resolveLang(langFlag, path string) (lang.Analyzer, string, error) {
	pick := strings.ToLower(langFlag)
	if pick == "" {
		pick = "auto"
	}
	if pick == "auto" {
		if path == "" || path == "-" {
			return nil, "text", nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if a, ok := lang.ByExtension(ext); ok {
			return a, a.Name(), nil
		}
		return nil, "text", nil
	}
	a, ok := lang.ByName(pick)
	if !ok {
		return nil, "", fmt.Errorf("invalid --lang value %q (want auto|text|markdown|html|jsx|tsx|ts|js)", pick)
	}
	if a == nil {
		// "text" resolves to no analyzer; still a valid selection.
		return nil, "text", nil
	}
	return a, a.Name(), nil
}

// llmEffort is the canonical effort level for the LLM passes.
type llmEffort string

const (
	effortOff  llmEffort = "off"
	effortLow  llmEffort = "low"
	effortHigh llmEffort = "high"
)

// pluginEnvActive reports whether the process is running under a Claude Code
// or Cursor plugin invocation. Both products export PLUGIN_ROOT env vars
// into their tool subshells; we use their presence as a signal that the
// user's Claude subscription is reachable via the `claude` CLI.
func pluginEnvActive() bool {
	return os.Getenv("CLAUDE_PLUGIN_ROOT") != "" || os.Getenv("CURSOR_PLUGIN_ROOT") != ""
}

// autoEnableLLM returns true when the LLM passes should be auto-enabled for
// this invocation: the plugin environment is present AND the `claude`
// binary is actually on $PATH. Both conditions are required so that running
// the CLI outside a plugin (or in a plugin environment that lacks the
// subscription binary) never silently burns API credits.
func autoEnableLLM(claudeBin string) bool {
	if !pluginEnvActive() {
		return false
	}
	if _, err := exec.LookPath(claudeBin); err != nil {
		return false
	}
	return true
}

// resolveEffort picks the LLM effort level for this run. Precedence:
//  1. --llm-effort when explicitly set (authoritative);
//  2. --llm-deep (sugar alias: true→high, false→off), over
//  3. --llm        (sugar alias: true→low,  false→off);
//  4. no explicit flags → auto (plugin-aware default).
//
// The second return value is true when the chosen effort came from the
// auto-default — that's what distinguishes fail-open (auto) from
// fail-closed (explicit) error handling for the LLM passes.
func resolveEffort(cmd *cobra.Command, effortFlag string, llmFlag, deepFlag bool, claudeBin string) (llmEffort, bool, error) {
	if cmd.Flags().Changed("llm-effort") {
		switch strings.ToLower(effortFlag) {
		case "off":
			return effortOff, false, nil
		case "low":
			return effortLow, false, nil
		case "high":
			return effortHigh, false, nil
		case "", "auto":
			return autoEffort(claudeBin), true, nil
		default:
			return effortOff, false, fmt.Errorf("invalid --llm-effort %q (want off|low|high|auto)", effortFlag)
		}
	}
	// --llm-deep is more specific than --llm, so let it win.
	if cmd.Flags().Changed("llm-deep") {
		if deepFlag {
			return effortHigh, false, nil
		}
		return effortOff, false, nil
	}
	if cmd.Flags().Changed("llm") {
		if llmFlag {
			return effortLow, false, nil
		}
		return effortOff, false, nil
	}
	return autoEffort(claudeBin), true, nil
}

func autoEffort(claudeBin string) llmEffort {
	if autoEnableLLM(claudeBin) {
		return effortHigh
	}
	return effortOff
}

func newCheckCmd() *cobra.Command {
	var (
		llmFlag   bool
		deepFlag  bool
		effort    string
		sentModel string
		docModel  string
		sentTO    time.Duration
		docTO     time.Duration
		langMode  string
	)
	cmd := &cobra.Command{
		Use:   "check [path|-]",
		Short: "Run detectors over a file (or stdin) and emit a JSON report.",
		Long: `Runs all 35 client-side detectors by default. Two optional LLM passes
run via the claude CLI:

  low   sentence-tier semantic analysis (Claude Haiku)
  high  low + document-tier structural analysis (Claude Sonnet)

Choose one with --llm-effort (off|low|high|auto), or use the sugar aliases:
  --llm       → --llm-effort=low
  --llm-deep  → --llm-effort=high

Under a Claude Code or Cursor plugin (detected via CLAUDE_PLUGIN_ROOT /
CURSOR_PLUGIN_ROOT) and when the claude CLI is on $PATH, --llm-effort=auto
resolves to 'high'; otherwise 'off'. Auto-enabled passes fail open (the
failure is reported under 'llm.<tier>.error' and the client-side results
are still returned); explicit passes propagate the error as exit code 3.

Input is taken from the path argument, or from stdin when the path is "-"
or omitted.

Language-aware mode masks non-prose regions of the input before detectors
run, so slop-cop flags prose only — not code, tags, URLs, or other syntax.
Pick a mode with --lang:

  auto      (default) pick from the file extension; "text" for stdin.
  text      no masking; treat input as plain prose.
  markdown  CommonMark; mask code fences, links, HTML, YAML front matter.
  html      HTML; mask tags, attributes, <script>/<style>/<pre>/<code>.
  jsx,tsx   JS/TS with JSX; keep comments, strings, template quasis, JSX text.
  ts,js     JS/TS without JSX.

Suppressions inside masked modes drop structural false positives
(e.g. 'dramatic-fragment' on headings, 'staccato-burst' across list items).`,
		Example: `  slop-cop check article.md --pretty
  cat article.md | slop-cop check --llm-effort=off
  slop-cop check component.tsx --lang=tsx --llm-effort=off
  slop-cop check - --lang=markdown --llm-effort=high < article.md`,
		Args: cobra.MaximumNArgs(1),
	}
	claudeBin := addClaudeBinFlag(cmd)
	pretty := addPrettyFlag(cmd)
	cmd.Flags().StringVar(&effort, "llm-effort", "auto", "LLM analysis effort: off|low|high|auto. Auto = high under plugin context, off otherwise.")
	cmd.Flags().BoolVar(&llmFlag, "llm", false, "Alias for --llm-effort=low (sentence tier via Claude Haiku).")
	cmd.Flags().BoolVar(&deepFlag, "llm-deep", false, "Alias for --llm-effort=high (sentence + document tiers, Haiku + Sonnet).")
	cmd.Flags().StringVar(&sentModel, "sentence-model", llm.DefaultSentenceModel, "Model slug for the sentence pass.")
	cmd.Flags().StringVar(&docModel, "document-model", llm.DefaultDocumentModel, "Model slug for the document pass.")
	cmd.Flags().DurationVar(&sentTO, "sentence-timeout", llm.DefaultSentenceTimeout, "Timeout for each sentence-pass chunk.")
	cmd.Flags().DurationVar(&docTO, "document-timeout", llm.DefaultDocumentTimeout, "Timeout for the document pass.")
	cmd.Flags().StringVar(&langMode, "lang", "auto", "Input language: auto|text|markdown|html|jsx|tsx|ts|js.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		path := pathArg(args)
		text, err := readInput(path)
		if err != nil {
			return err
		}
		ctx := runContext(cmd)

		analyzer, langName, err := resolveLang(langMode, path)
		if err != nil {
			return usageError{err: err}
		}

		eff, auto, err := resolveEffort(cmd, effort, llmFlag, deepFlag, *claudeBin)
		if err != nil {
			return usageError{err: err}
		}
		runSentence := eff == effortLow || eff == effortHigh
		runDocument := eff == effortHigh

		// When a lang analyzer is selected, detectors (and the LLM passes)
		// run on a masked copy of the input. Offsets still index into the
		// original bytes (Analyze preserves length), so we re-slice
		// MatchedText from the original in ApplySuppressions.
		scanText := text
		var suppress []lang.Range
		if analyzer != nil {
			m, s, aerr := analyzer.Analyze(text)
			if aerr != nil {
				return fmt.Errorf("%s: analyze: %w", analyzer.Name(), aerr)
			}
			scanText, suppress = m, s
		}

		violations := detectors.RunClient(scanText)

		var llmRep *llmReport
		ensureReport := func() *llmReport {
			if llmRep == nil {
				llmRep = &llmReport{}
			}
			return llmRep
		}

		if runSentence {
			opts := llm.Options{Bin: *claudeBin, Model: sentModel, Timeout: sentTO}
			vs, err := llm.RunSentence(ctx, scanText, opts)
			if err != nil {
				if auto {
					fmt.Fprintln(os.Stderr, "slop-cop: sentence LLM pass skipped (auto-enabled, claude failed):", err)
					ensureReport().Sentence = &llmPassStatus{Auto: true, Ran: false, Error: err.Error()}
				} else {
					return llmError{err: fmt.Errorf("sentence pass: %w", err)}
				}
			} else {
				violations = append(violations, vs...)
				ensureReport().Sentence = &llmPassStatus{Auto: auto, Ran: true}
			}
		}
		if runDocument {
			opts := llm.Options{Bin: *claudeBin, Model: docModel, Timeout: docTO}
			vs, err := llm.RunDocument(ctx, scanText, opts)
			if err != nil {
				if auto {
					fmt.Fprintln(os.Stderr, "slop-cop: document LLM pass skipped (auto-enabled, claude failed):", err)
					ensureReport().Document = &llmPassStatus{Auto: true, Ran: false, Error: err.Error()}
				} else {
					return llmError{err: fmt.Errorf("document pass: %w", err)}
				}
			} else {
				violations = append(violations, vs...)
				ensureReport().Document = &llmPassStatus{Auto: auto, Ran: true}
			}
		}
		if runSentence || runDocument {
			violations = detectors.Deduplicate(violations)
		}

		if analyzer != nil {
			violations = analyzer.ApplySuppressions(violations, suppress, text)
		}

		sortViolations(violations)
		if violations == nil {
			violations = []types.Violation{}
		}

		report := checkReport{
			TextLength:       len(text),
			Violations:       violations,
			CountsByRule:     map[string]int{},
			CountsByCategory: map[types.ViolationCategory]int{},
			Lang:             langName,
			LLMEffort:        string(eff),
			LLM:              llmRep,
		}
		for _, v := range violations {
			report.CountsByRule[v.RuleID]++
			if rule, ok := rules.ByID[v.RuleID]; ok {
				report.CountsByCategory[rule.Category]++
			}
		}
		return writeJSON(report, *pretty)
	}
	return cmd
}
