package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yasyf/slop-cop/internal/llm"
	"github.com/yasyf/slop-cop/internal/rules"
)

// rewriteReport is the JSON document returned by `slop-cop rewrite`.
type rewriteReport struct {
	Rewritten   string   `json:"rewritten"`
	AppliedRules []string `json:"applied_rules"`
}

func newRewriteCmd() *cobra.Command {
	var (
		ruleList  []string
		claudeBin string
		model     string
		timeout   time.Duration
		pretty    bool
	)
	cmd := &cobra.Command{
		Use:   "rewrite [path|-]",
		Short: "Rewrite a paragraph via `claude -p`, applying the slop-cop rewrite rules.",
		Long: `Sends the input text to the claude CLI with the slop-cop rewrite system prompt
(default rule directives + any rule hints provided via --rules) and prints a
JSON document containing the rewritten text.`,
		Example: `  slop-cop rewrite draft.txt
  slop-cop rewrite - --rules filler-adverbs,hedge-stack < draft.txt`,
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

			var hints, applied []string
			for _, id := range ruleList {
				id = strings.TrimSpace(id)
				if id == "" {
					continue
				}
				r, ok := rules.ByID[id]
				if !ok {
					return usageError{err: fmt.Errorf("unknown rule id: %s", id)}
				}
				if r.RewriteHint != "" {
					hints = append(hints, r.RewriteHint)
					applied = append(applied, id)
				}
			}

			opts := llm.Options{Bin: claudeBin, Model: model, Timeout: timeout}
			rewritten, err := llm.RewriteParagraph(ctx, text, hints, opts)
			if err != nil {
				return llmError{err: err}
			}
			return writeJSON(rewriteReport{
				Rewritten:    rewritten,
				AppliedRules: applied,
			}, pretty)
		},
	}
	cmd.Flags().StringSliceVar(&ruleList, "rules", nil, "Comma-separated rule IDs whose rewrite hints should be added to the prompt.")
	cmd.Flags().StringVar(&claudeBin, "claude-bin", "claude", "Path to the claude CLI binary.")
	cmd.Flags().StringVar(&model, "model", llm.DefaultRewriteModel, "Model slug for the rewrite call.")
	cmd.Flags().DurationVar(&timeout, "timeout", llm.DefaultRewriteTimeout, "Timeout for the rewrite call.")
	cmd.Flags().BoolVar(&pretty, "pretty", false, "Indent JSON output.")
	return cmd
}
