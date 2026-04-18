package main

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"
	"github.com/yasyf/slop-cop/internal/detectors"
	"github.com/yasyf/slop-cop/internal/llm"
	"github.com/yasyf/slop-cop/internal/rules"
	"github.com/yasyf/slop-cop/internal/types"
)

// checkReport is the JSON document emitted by `slop-cop check`. Counts are
// denormalised to save callers a pass over the violations slice.
type checkReport struct {
	TextLength       int                           `json:"text_length"`
	Violations       []types.Violation             `json:"violations"`
	CountsByRule     map[string]int                `json:"counts_by_rule"`
	CountsByCategory map[types.ViolationCategory]int `json:"counts_by_category"`
}

func newCheckCmd() *cobra.Command {
	var (
		useLLM     bool
		useDeep    bool
		claudeBin  string
		sentModel  string
		docModel   string
		sentTO     time.Duration
		docTO      time.Duration
		pretty     bool
	)
	cmd := &cobra.Command{
		Use:   "check [path|-]",
		Short: "Run detectors over a file (or stdin) and emit a JSON report.",
		Long: `Runs all 35 client-side detectors by default. Pass --llm to add the
sentence-level semantic pass (Claude Haiku via the claude CLI); pass
--llm-deep to add the document-level structural pass (Claude Sonnet).

Input is taken from the path argument, or from stdin when the path is "-"
or omitted.`,
		Example: `  slop-cop check article.md --pretty
  cat article.md | slop-cop check --llm
  slop-cop check - --llm --llm-deep < article.md`,
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

			violations := detectors.RunClient(text)

			if useLLM || useDeep {
				opts := llm.Options{Bin: claudeBin}
				if useLLM {
					sentOpts := opts
					sentOpts.Model = sentModel
					sentOpts.Timeout = sentTO
					vs, err := llm.RunSentence(ctx, text, sentOpts)
					if err != nil {
						return llmError{err: fmt.Errorf("sentence pass: %w", err)}
					}
					violations = append(violations, vs...)
				}
				if useDeep {
					docOpts := opts
					docOpts.Model = docModel
					docOpts.Timeout = docTO
					vs, err := llm.RunDocument(ctx, text, docOpts)
					if err != nil {
						return llmError{err: fmt.Errorf("document pass: %w", err)}
					}
					violations = append(violations, vs...)
				}
				violations = detectors.Deduplicate(violations)
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
	return cmd
}
