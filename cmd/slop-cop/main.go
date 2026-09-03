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
	exitOK         = 0
	exitViolations = 1
	exitIO         = 2
	exitLLM        = 3
	exitUsage      = 4
)

// errViolationsFound signals a completed check whose report carries at least
// one violation. Only --strict raises it; the report on stdout is the message,
// so main exits on it without writing a diagnostic.
var errViolationsFound = errors.New("violations found")

// usageError wraps a flag/argument validation problem so we can map it to
// exit code 4 at the top level.
type usageError struct{ err error }

func (u usageError) Error() string { return u.err.Error() }
func (u usageError) Unwrap() error { return u.err }

// llmError flags failures that originated in the LLM subprocess layer.
type llmError struct{ err error }

func (l llmError) Error() string { return l.err.Error() }
func (l llmError) Unwrap() error { return l.err }

func main() {
	cmd := newRoot()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	err := cmd.Execute()
	if err != nil && !errors.Is(err, errViolationsFound) {
		fmt.Fprintln(os.Stderr, "slop-cop:", err)
	}
	os.Exit(exitCodeFor(err))
}

// exitCodeFor maps a command's terminating error onto the documented exit
// codes. errViolationsFound is a completed run, not a failure.
func exitCodeFor(err error) int {
	switch {
	case err == nil:
		return exitOK
	case errors.Is(err, errViolationsFound):
		return exitViolations
	case errors.As(err, new(usageError)):
		return exitUsage
	case errors.As(err, new(llmError)):
		return exitLLM
	default:
		return exitIO
	}
}

func newRoot() *cobra.Command {
	ver, _ := buildMetadata()
	root := &cobra.Command{
		Use:     "slop-cop",
		Short:   "Detect LLM-generated prose patterns; emit structured JSON.",
		Version: ver,
		Long: `slop-cop runs regex + structural detectors (and optional Codex-backed
semantic analysis) over a piece of text, and prints a JSON report aimed at
other agents.

Input is read from the file argument or from stdin when the argument is "-"
or omitted. Output is JSON on stdout; errors go to stderr.

Exit codes:
  0  success, including a run that found violations
  1  check --strict completed and the report carries violations
  2  input/IO error
  3  LLM subprocess error
  4  flag/usage error`,
	}
	// --version prints the bare stamped version, one line, no decoration:
	// the plugin installer compares it against the release tag.
	root.SetVersionTemplate("{{.Version}}\n")
	root.AddCommand(newCheckCmd())
	root.AddCommand(newPlainifyCmd())
	root.AddCommand(newRewriteCmd())
	root.AddCommand(newRulesCmd())
	root.AddCommand(newVersionCmd())
	return root
}
