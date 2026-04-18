// Command slop-cop is a CLI for detecting LLM-generated prose patterns in
// text. It is designed for agent consumption: commands are non-interactive,
// output is structured JSON on stdout, and diagnostics go to stderr.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// exit codes
const (
	exitOK        = 0
	exitIO        = 2
	exitLLM       = 3
	exitUsage     = 4
)

// usageError wraps a flag/argument validation problem so we can map it to
// exit code 4 at the top level.
type usageError struct{ err error }

func (u usageError) Error() string { return u.err.Error() }
func (u usageError) Unwrap() error { return u.err }

// llmError flags failures that originated in the `claude` subprocess layer.
type llmError struct{ err error }

func (l llmError) Error() string { return l.err.Error() }
func (l llmError) Unwrap() error { return l.err }

func main() {
	cmd := newRoot()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "slop-cop:", err)
		switch {
		case errors.As(err, new(usageError)):
			os.Exit(exitUsage)
		case errors.As(err, new(llmError)):
			os.Exit(exitLLM)
		default:
			os.Exit(exitIO)
		}
	}
	os.Exit(exitOK)
}

func newRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "slop-cop",
		Short: "Detect LLM-generated prose patterns; emit structured JSON.",
		Long: `slop-cop runs regex + structural detectors (and optional Claude-backed
semantic analysis) over a piece of text, and prints a JSON report aimed at
other agents.

Input is read from the file argument or from stdin when the argument is "-"
or omitted. Output is JSON on stdout; errors go to stderr.

Exit codes:
  0  success (including "no violations found")
  2  input/IO error
  3  claude subprocess error
  4  flag/usage error`,
	}
	root.AddCommand(newCheckCmd())
	root.AddCommand(newRewriteCmd())
	root.AddCommand(newRulesCmd())
	root.AddCommand(newVersionCmd())
	return root
}
