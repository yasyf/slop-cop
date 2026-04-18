package main

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
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
// denormalised to save callers a pass over the violations slice.
type checkReport struct {
	TextLength       int                             `json:"text_length"`
	Violations       []types.Violation               `json:"violations"`
	CountsByRule     map[string]int                  `json:"counts_by_rule"`
	CountsByCategory map[types.ViolationCategory]int `json:"counts_by_category"`
	Markdown         bool                            `json:"markdown,omitempty"`
}

// markdownExts enumerates file extensions that trigger markdown mode when
// --markdown=auto (the default). Matched case-insensitively.
var markdownExts = map[string]bool{
	".md":       true,
	".markdown": true,
	".mdx":      true,
}

func resolveMarkdown(mode, path string) (bool, error) {
	switch strings.ToLower(mode) {
	case "on", "true", "yes":
		return true, nil
	case "off", "false", "no":
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
		claudeBin string
		sentModel string
		docModel  string
		sentTO    time.Duration
		docTO     time.Duration
		pretty    bool
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
		RunE: func(cmd *cobra.Command, args []string) error {
			path := ""
			if len(args) == 1 {
				path = args[0]
			}
			text, err := readInput(path)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			mdOn, err := resolveMarkdown(mdMode, path)
			if err != nil {
				return usageError{err: err}
			}

			// In markdown mode we run detectors (and the LLM passes) on a
			// masked copy of the input. Violation byte offsets still index
			// into the original bytes (mask preserves length); we re-slice
			// matchedText from the original after filtering.
			scanText := text
			var suppress []markdown.Range
			if mdOn {
				scanText, suppress, _ = markdown.Analyze(text)
			}

			violations := detectors.RunClient(scanText)

			if useLLM || useDeep {
				opts := llm.Options{Bin: claudeBin}
				if useLLM {
					sentOpts := opts
					sentOpts.Model = sentModel
					sentOpts.Timeout = sentTO
					vs, err := llm.RunSentence(ctx, scanText, sentOpts)
					if err != nil {
						return llmError{err: fmt.Errorf("sentence pass: %w", err)}
					}
					violations = append(violations, vs...)
				}
				if useDeep {
					docOpts := opts
					docOpts.Model = docModel
					docOpts.Timeout = docTO
					vs, err := llm.RunDocument(ctx, scanText, docOpts)
					if err != nil {
						return llmError{err: fmt.Errorf("document pass: %w", err)}
					}
					violations = append(violations, vs...)
				}
				violations = detectors.Deduplicate(violations)
			}

			if mdOn {
				violations = applyMarkdownSuppressions(violations, suppress, text)
			}

			sort.SliceStable(violations, func(i, j int) bool {
				if violations[i].StartIndex != violations[j].StartIndex {
					return violations[i].StartIndex < violations[j].StartIndex
				}
				if violations[i].EndIndex != violations[j].EndIndex {
					return violations[i].EndIndex < violations[j].EndIndex
				}
				return violations[i].RuleID < violations[j].RuleID
			})

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
			return writeJSON(report, pretty)
		},
	}
	cmd.Flags().BoolVar(&useLLM, "llm", false, "Run the sentence-level semantic pass via `claude -p`.")
	cmd.Flags().BoolVar(&useDeep, "llm-deep", false, "Run the document-level structural pass via `claude -p`.")
	cmd.Flags().StringVar(&claudeBin, "claude-bin", "claude", "Path to the claude CLI binary.")
	cmd.Flags().StringVar(&sentModel, "sentence-model", llm.DefaultSentenceModel, "Model slug for the sentence pass.")
	cmd.Flags().StringVar(&docModel, "document-model", llm.DefaultDocumentModel, "Model slug for the document pass.")
	cmd.Flags().DurationVar(&sentTO, "sentence-timeout", llm.DefaultSentenceTimeout, "Timeout for each sentence-pass chunk.")
	cmd.Flags().DurationVar(&docTO, "document-timeout", llm.DefaultDocumentTimeout, "Timeout for the document pass.")
	cmd.Flags().BoolVar(&pretty, "pretty", false, "Indent JSON output.")
	cmd.Flags().StringVar(&mdMode, "markdown", "auto", "Treat input as markdown: auto|on|off.")
	return cmd
}

// applyMarkdownSuppressions drops violations that correspond to false
// positives on markdown structural elements. It runs *after* the detector
// passes on the masked text and *before* sorting/reporting. Surviving
// violations have their MatchedText re-populated from the original input so
// consumers see actual prose bytes, not the masked whitespace.
func applyMarkdownSuppressions(vs []types.Violation, suppress []markdown.Range, original string) []types.Violation {
	out := vs[:0]
	for _, v := range vs {
		switch v.RuleID {
		case "dramatic-fragment":
			if markdown.Overlaps(v.StartIndex, v.EndIndex, suppress, markdown.KindHeading) {
				continue
			}
		case "staccato-burst":
			// A short-sentence burst that straddles two or more consecutive
			// list items is just a list; drop it.
			if markdown.CountOverlapping(v.StartIndex, v.EndIndex, suppress, markdown.KindListItem) >= 2 {
				continue
			}
		}
		if v.StartIndex >= 0 && v.EndIndex <= len(original) && v.EndIndex >= v.StartIndex {
			v.MatchedText = original[v.StartIndex:v.EndIndex]
		}
		out = append(out, v)
	}
	// Copy to release aliased capacity and give callers a clean slice.
	result := make([]types.Violation, len(out))
	copy(result, out)
	return result
}
