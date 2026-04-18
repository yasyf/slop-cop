package main

import (
	"context"
	"sort"

	"github.com/spf13/cobra"
	"github.com/yasyf/slop-cop/internal/types"
)

// pathArg extracts the optional `[path|-]` argument common to check and
// rewrite. Returns "" when no path was given (meaning read stdin).
func pathArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

// runContext returns the command's attached context, falling back to a
// fresh background context if none is set. Cobra's ExecuteContext wires one
// in automatically in normal invocations; the fallback keeps unit tests
// that construct commands by hand working.
func runContext(cmd *cobra.Command) context.Context {
	if ctx := cmd.Context(); ctx != nil {
		return ctx
	}
	return context.Background()
}

// addPrettyFlag wires the shared --pretty flag onto cmd and returns a
// pointer the caller can read from RunE.
func addPrettyFlag(cmd *cobra.Command) *bool {
	var pretty bool
	cmd.Flags().BoolVar(&pretty, "pretty", false, "Indent JSON output.")
	return &pretty
}

// addClaudeBinFlag wires the shared --claude-bin flag onto cmd.
func addClaudeBinFlag(cmd *cobra.Command) *string {
	var bin string
	cmd.Flags().StringVar(&bin, "claude-bin", "claude", "Path to the claude CLI binary.")
	return &bin
}

// sortViolations orders violations deterministically by (start, end, rule)
// so consumers get stable JSON regardless of which detector produced the
// hit. Sort is stable so ties preserve input order.
func sortViolations(vs []types.Violation) {
	sort.SliceStable(vs, func(i, j int) bool {
		if vs[i].StartIndex != vs[j].StartIndex {
			return vs[i].StartIndex < vs[j].StartIndex
		}
		if vs[i].EndIndex != vs[j].EndIndex {
			return vs[i].EndIndex < vs[j].EndIndex
		}
		return vs[i].RuleID < vs[j].RuleID
	})
}
