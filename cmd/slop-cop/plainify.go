package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/spf13/cobra"
	"github.com/yasyf/slop-cop/internal/llm"
)

// plainifyResult is the JSON document `slop-cop plainify` returns for one
// piece of prose.
type plainifyResult struct {
	Plain      string               `json:"plain"`
	Words      int                  `json:"words"`
	Truncated  bool                 `json:"truncated"`
	Violations []llm.PlainViolation `json:"violations"`
}

// plainifyEntryResult is one element of the array --json prints, carrying the
// id its entry arrived with.
type plainifyEntryResult struct {
	ID string `json:"id"`
	plainifyResult
}

func newPlainifyCmd() *cobra.Command {
	var (
		batch       bool
		maxWords    int
		forbid      []string
		nameByTitle bool
		glossary    string
		model       string
		timeout     time.Duration
	)
	cmd := &cobra.Command{
		Use:   "plainify [path|-]",
		Short: "Rewrite prose into plain English via `claude -p`.",
		Long: `Sends the input text to the claude CLI with the plainify system prompt and
prints a JSON document holding the plain-English rewrite.

The contract is the same every call: keep every fact, name, number, and file
path; short sentences and everyday words; fenced and inline code left
unchanged; the rewrite alone, with no preamble or labels.

--max-words and --forbid reach the model as instructions and are graded again
after it answers. A rewrite that misses either is retried once with its misses
named; what the retry still misses is reported in "truncated" and "violations"
rather than dropped.

--json reads a JSON array of {"id","text"} entries instead of one piece of
prose, runs one call per entry, and prints one result per entry in the order
they arrived.`,
		Example: `  slop-cop plainify draft.md
  slop-cop plainify - --max-words 120 < draft.md
  slop-cop plainify findings.json --json --forbid '\b(DQ|A|Q|V)\d+\b' --name-by-title --glossary titles.json`,
		Args: cobra.MaximumNArgs(1),
	}
	pretty := addPrettyFlag(cmd)
	cmd.Flags().BoolVar(&batch, "json", false, `Read a JSON array of {"id","text"} entries and emit one result per entry.`)
	cmd.Flags().IntVar(&maxWords, "max-words", 0, "Word budget for the rewrite; 0 sets none.")
	cmd.Flags().StringArrayVar(&forbid, "forbid", nil, "Regular expression the rewrite must not match; repeatable.")
	cmd.Flags().BoolVar(&nameByTitle, "name-by-title", false, "Name referenced items by their titles rather than their identifiers.")
	cmd.Flags().StringVar(&glossary, "glossary", "", `Path to a JSON object mapping identifier to title, e.g. {"DQ4":"Worker transport"}.`)
	cmd.Flags().StringVar(&model, "model", llm.DefaultRewriteModel, "Claude model slug for the plainify call.")
	cmd.Flags().DurationVar(&timeout, "timeout", llm.DefaultRewriteTimeout, "Wall-clock bound on each plainify call, retries included.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		text, err := readInput(pathArg(args))
		if err != nil {
			return err
		}
		constraints, err := resolvePlainifyConstraints(maxWords, forbid, nameByTitle, glossary)
		if err != nil {
			return err
		}
		ctx := runContext(cmd)
		opts := llm.Options{Model: model, Timeout: timeout}

		if !batch {
			res, err := llm.Plainify(ctx, text, constraints, opts)
			if err != nil {
				return llmError{err: err}
			}
			return writeJSON(newPlainifyResult(res), *pretty)
		}

		var entries []llm.PlainEntry
		if err := json.Unmarshal([]byte(text), &entries); err != nil {
			return fmt.Errorf("decoding --json entries: %w", err)
		}
		// A JSON null decodes to a nil slice without erroring; an empty array
		// decodes non-nil and is a real answer.
		if entries == nil {
			return fmt.Errorf(`decoding --json entries: want an array of {"id","text"}, got null`)
		}
		results, err := llm.PlainifyBatch(ctx, entries, constraints, opts)
		if err != nil {
			return llmError{err: err}
		}
		out := make([]plainifyEntryResult, len(results))
		for i, res := range results {
			out[i] = plainifyEntryResult{ID: entries[i].ID, plainifyResult: newPlainifyResult(res)}
		}
		return writeJSON(out, *pretty)
	}
	return cmd
}

func newPlainifyResult(res llm.PlainResult) plainifyResult {
	return plainifyResult{
		Plain:      res.Plain,
		Words:      res.Words,
		Truncated:  res.Truncated,
		Violations: res.Violations,
	}
}

// resolvePlainifyConstraints compiles the constraint flags into the shape the
// llm layer takes, mapping a bad pattern or budget to a usage error and an
// unreadable glossary to an IO error.
func resolvePlainifyConstraints(maxWords int, forbid []string, nameByTitle bool, glossaryPath string) (llm.PlainifyConstraints, error) {
	c := llm.PlainifyConstraints{MaxWords: maxWords, NameByTitle: nameByTitle}
	if maxWords < 0 {
		return c, usageError{err: fmt.Errorf("--max-words must not be negative: %d", maxWords)}
	}
	c.Forbid = make([]*regexp.Regexp, 0, len(forbid))
	for _, pattern := range forbid {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return c, usageError{err: fmt.Errorf("--forbid %s: %w", pattern, err)}
		}
		c.Forbid = append(c.Forbid, re)
	}
	if glossaryPath == "" {
		return c, nil
	}
	raw, err := os.ReadFile(glossaryPath) //nolint:gosec // G304: path is the user-supplied --glossary argument this command is designed to read.
	if err != nil {
		return c, fmt.Errorf("reading %s: %w", glossaryPath, err)
	}
	if err := json.Unmarshal(raw, &c.Glossary); err != nil {
		return c, fmt.Errorf("decoding %s: %w", glossaryPath, err)
	}
	return c, nil
}
