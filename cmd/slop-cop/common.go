package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

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

// outputFormat is the rendering `check` uses for its report.
type outputFormat string

const (
	formatJSON    outputFormat = "json"
	formatCompact outputFormat = "compact"
)

// addFormatFlag wires the shared --format flag onto cmd and returns a
// pointer the caller can read from RunE.
func addFormatFlag(cmd *cobra.Command) *string {
	var format string
	cmd.Flags().StringVar(&format, "format", string(formatJSON), "Output format: json (the full report) or compact (one tab-separated line per violation, then a counts line).")
	return &format
}

// resolveFormat maps --format onto a rendering. An unknown value is a usage
// error.
func resolveFormat(formatFlag string) (outputFormat, error) {
	pick := outputFormat(strings.ToLower(formatFlag))
	if pick == "" {
		return formatJSON, nil
	}
	switch pick {
	case formatJSON, formatCompact:
		return pick, nil
	}
	return "", fmt.Errorf("invalid --format value %q (want json|compact)", formatFlag)
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
