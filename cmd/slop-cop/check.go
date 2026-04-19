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
	"github.com/yasyf/slop-cop/internal/llm"
	"github.com/yasyf/slop-cop/internal/markdown"
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
	// Markdown reports whether markdown-aware masking + suppression ran.
	// Always emitted (not omitempty) so agents can tell "off" from "absent".
	Markdown bool `json:"markdown"`
	// LLM reports what the sentence/document-tier LLM passes actually did,
	// including whether they were auto-enabled from the plugin environment
	// and whether they failed. Omitted entirely when no LLM pass was
	// requested or auto-enabled.
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
	// Auto is true when the pass was enabled by auto-default (plugin env
	// detected + claude CLI on $PATH) rather than an explicit --llm flag.
	Auto bool `json:"auto"`
	// Ran is true when the pass executed successfully and contributed
	// violations to the report.
	Ran bool `json:"ran"`
	// Error holds the pass's error message when Ran is false. Empty when
	// the pass succeeded or wasn't attempted.
	Error string `json:"error,omitempty"`
}

// markdownExts enumerates file extensions that activate markdown mode under
// --markdown=auto. Matched case-insensitively.
var markdownExts = map[string]bool{
	".md":       true,
	".markdown": true,
	".mdx":      true,
}

// resolveMarkdown interprets the value of --markdown. The three accepted
// values are strict: auto, on, off. Any other value is a usage error.
func resolveMarkdown(mode, path string) (bool, error) {
	switch strings.ToLower(mode) {
	case "on":
		return true, nil
	case "off":
		return false, nil
	case "", "auto":
		if path == "" || path == "-" {
			return false, nil
		}
		return markdownExts[strings.ToLower(filepath.Ext(path))], nil
	default:
		return false, fmt.Errorf("invalid --markdown value %q (want auto|on|off)", mode)
	}
}

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

func newCheckCmd() *cobra.Command {
	var (
		useLLM    bool
		useDeep   bool
		sentModel string
		docModel  string
		sentTO    time.Duration
		docTO     time.Duration
		mdMode    string
	)
	cmd := &cobra.Command{
		Use:   "check [path|-]",
		Short: "Run detectors over a file (or stdin) and emit a JSON report.",
		Long: `Runs all 35 client-side detectors by default. The sentence-level
(Claude Haiku) and document-level (Claude Sonnet) semantic passes run via
the claude CLI and can be toggled with --llm / --llm-deep.

When invoked from a Claude Code or Cursor plugin (detected via the
CLAUDE_PLUGIN_ROOT / CURSOR_PLUGIN_ROOT env vars) and the claude CLI is on
$PATH, both LLM passes default to on. Pass --llm=false / --llm-deep=false
to opt out. If an auto-enabled pass fails for any reason (missing auth,
network error, rate limit) the failure is reported in the JSON under
'llm.<tier>.error' and the client-side detector results are still
returned; a user-requested pass with --llm=true propagates the error as
exit code 3.

Input is taken from the path argument, or from stdin when the path is "-"
or omitted.

Markdown-aware mode masks non-prose regions (code blocks, inline code, link
and image destinations, autolinks, HTML, YAML front matter) before running
the detectors, and suppresses a few structural false positives
(e.g. 'dramatic-fragment' on ATX/setext headings, 'staccato-burst' across
bulleted-list items). Activation is controlled by --markdown:

  auto   On when the input path ends in .md / .markdown / .mdx (default).
         Off for stdin.
  on     Force markdown mode.
  off    Treat the input as plain text regardless of extension.`,
		Example: `  slop-cop check article.md --pretty
  cat article.md | slop-cop check --llm=false
  slop-cop check - --markdown=on --llm --llm-deep < article.md`,
		Args: cobra.MaximumNArgs(1),
	}
	claudeBin := addClaudeBinFlag(cmd)
	pretty := addPrettyFlag(cmd)
	cmd.Flags().BoolVar(&useLLM, "llm", false, "Run the sentence-level semantic pass via `claude -p`. Default auto-on under a Claude Code / Cursor plugin; pass --llm=false to opt out.")
	cmd.Flags().BoolVar(&useDeep, "llm-deep", false, "Run the document-level structural pass via `claude -p`. Default auto-on under a Claude Code / Cursor plugin; pass --llm-deep=false to opt out.")
	cmd.Flags().StringVar(&sentModel, "sentence-model", llm.DefaultSentenceModel, "Model slug for the sentence pass.")
	cmd.Flags().StringVar(&docModel, "document-model", llm.DefaultDocumentModel, "Model slug for the document pass.")
	cmd.Flags().DurationVar(&sentTO, "sentence-timeout", llm.DefaultSentenceTimeout, "Timeout for each sentence-pass chunk.")
	cmd.Flags().DurationVar(&docTO, "document-timeout", llm.DefaultDocumentTimeout, "Timeout for the document pass.")
	cmd.Flags().StringVar(&mdMode, "markdown", "auto", "Treat input as markdown: auto|on|off.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		path := pathArg(args)
		text, err := readInput(path)
		if err != nil {
			return err
		}
		ctx := runContext(cmd)

		mdOn, err := resolveMarkdown(mdMode, path)
		if err != nil {
			return usageError{err: err}
		}

		// Auto-enable LLM passes in plugin contexts the user didn't already
		// address. autoSentence / autoDocument track "was this flag set by
		// the auto-default?" so we can degrade gracefully on failure.
		autoSentence, autoDocument := false, false
		if !cmd.Flags().Changed("llm") && !useLLM && autoEnableLLM(*claudeBin) {
			useLLM, autoSentence = true, true
		}
		if !cmd.Flags().Changed("llm-deep") && !useDeep && autoEnableLLM(*claudeBin) {
			useDeep, autoDocument = true, true
		}

		// In markdown mode, detectors (and the LLM passes) run on a masked
		// copy of the input. Offsets still index into the original bytes
		// (Analyze preserves length), so we re-slice MatchedText from the
		// original after filtering in ApplySuppressions.
		scanText := text
		var suppress []markdown.Range
		if mdOn {
			scanText, suppress, _ = markdown.Analyze(text)
		}

		violations := detectors.RunClient(scanText)

		var llmRep *llmReport
		ensureReport := func() *llmReport {
			if llmRep == nil {
				llmRep = &llmReport{}
			}
			return llmRep
		}

		if useLLM {
			opts := llm.Options{Bin: *claudeBin, Model: sentModel, Timeout: sentTO}
			vs, err := llm.RunSentence(ctx, scanText, opts)
			if err != nil {
				if autoSentence {
					fmt.Fprintln(os.Stderr, "slop-cop: sentence LLM pass skipped (auto-enabled, claude failed):", err)
					ensureReport().Sentence = &llmPassStatus{Auto: true, Ran: false, Error: err.Error()}
				} else {
					return llmError{err: fmt.Errorf("sentence pass: %w", err)}
				}
			} else {
				violations = append(violations, vs...)
				ensureReport().Sentence = &llmPassStatus{Auto: autoSentence, Ran: true}
			}
		}
		if useDeep {
			opts := llm.Options{Bin: *claudeBin, Model: docModel, Timeout: docTO}
			vs, err := llm.RunDocument(ctx, scanText, opts)
			if err != nil {
				if autoDocument {
					fmt.Fprintln(os.Stderr, "slop-cop: document LLM pass skipped (auto-enabled, claude failed):", err)
					ensureReport().Document = &llmPassStatus{Auto: true, Ran: false, Error: err.Error()}
				} else {
					return llmError{err: fmt.Errorf("document pass: %w", err)}
				}
			} else {
				violations = append(violations, vs...)
				ensureReport().Document = &llmPassStatus{Auto: autoDocument, Ran: true}
			}
		}
		if useLLM || useDeep {
			violations = detectors.Deduplicate(violations)
		}

		if mdOn {
			violations = markdown.ApplySuppressions(violations, suppress, text)
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
			Markdown:         mdOn,
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
