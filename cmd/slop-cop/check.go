package main

import (
	"fmt"
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
		Long: `Runs all 35 client-side detectors by default. Pass --llm to add the
sentence-level semantic pass (Claude Haiku via the claude CLI); pass
--llm-deep to add the document-level structural pass (Claude Sonnet).

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
  cat article.md | slop-cop check --llm
  slop-cop check - --markdown=on --llm --llm-deep < article.md`,
		Args: cobra.MaximumNArgs(1),
	}
	claudeBin := addClaudeBinFlag(cmd)
	pretty := addPrettyFlag(cmd)
	cmd.Flags().BoolVar(&useLLM, "llm", false, "Run the sentence-level semantic pass via `claude -p`.")
	cmd.Flags().BoolVar(&useDeep, "llm-deep", false, "Run the document-level structural pass via `claude -p`.")
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

		if useLLM {
			opts := llm.Options{Bin: *claudeBin, Model: sentModel, Timeout: sentTO}
			vs, err := llm.RunSentence(ctx, scanText, opts)
			if err != nil {
				return llmError{err: fmt.Errorf("sentence pass: %w", err)}
			}
			violations = append(violations, vs...)
		}
		if useDeep {
			opts := llm.Options{Bin: *claudeBin, Model: docModel, Timeout: docTO}
			vs, err := llm.RunDocument(ctx, scanText, opts)
			if err != nil {
				return llmError{err: fmt.Errorf("document pass: %w", err)}
			}
			violations = append(violations, vs...)
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
