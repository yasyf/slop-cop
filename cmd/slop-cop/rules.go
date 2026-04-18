package main

import (
	"github.com/spf13/cobra"
	"github.com/yasyf/slop-cop/internal/rules"
	"github.com/yasyf/slop-cop/internal/types"
)

func newRulesCmd() *cobra.Command {
	var (
		category string
		llmOnly  bool
		pretty   bool
	)
	cmd := &cobra.Command{
		Use:   "rules",
		Short: "Print the full rule catalogue as JSON.",
		Long:  `Emit metadata for every rule so agents can map rule IDs to categories, tips, and rewrite directives without embedding the list themselves.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			filtered := make([]types.ViolationRule, 0, len(rules.All))
			for _, r := range rules.All {
				if category != "" && string(r.Category) != category {
					continue
				}
				if llmOnly && !r.RequiresLLM {
					continue
				}
				filtered = append(filtered, r)
			}
			return writeJSON(map[string]any{
				"count": len(filtered),
				"rules": filtered,
			}, pretty)
		},
	}
	cmd.Flags().StringVar(&category, "category", "", "Filter by category (word-choice, sentence-structure, rhetorical, structural, framing).")
	cmd.Flags().BoolVar(&llmOnly, "llm-only", false, "Only show rules that require the LLM passes.")
	cmd.Flags().BoolVar(&pretty, "pretty", false, "Indent JSON output.")
	return cmd
}
