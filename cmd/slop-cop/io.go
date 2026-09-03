package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"unicode/utf8"
)

// readInput loads text from a file path (or stdin when path is "" or "-").
func readInput(path string) (string, error) {
	if path == "" || path == "-" {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		return string(b), nil
	}
	b, err := os.ReadFile(path) //nolint:gosec // G304: path is the user-supplied file argument this CLI is designed to read.
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}
	return string(b), nil
}

// writeJSON serialises v to stdout, optionally indented for human
// inspection. Always terminates with a newline.
func writeJSON(v any, pretty bool) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if pretty {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(v)
}

// writeCompact renders the report as one `ruleId<TAB>line:col<TAB>matchedText`
// line per violation, then a `counts<TAB>rule=n ...` line. Columns are 1-based
// runes; StartIndex is a UTF-8 byte offset, so text supplies the mapping.
func writeCompact(report checkReport, text string) error {
	w := bufio.NewWriter(os.Stdout)
	for _, v := range report.Violations {
		line, col := lineColumn(text, v.StartIndex)
		if _, err := fmt.Fprintf(w, "%s\t%d:%d\t%s\n", v.RuleID, line, col, compactEscape(v.MatchedText)); err != nil {
			return err
		}
	}
	ids := make([]string, 0, len(report.CountsByRule))
	for id := range report.CountsByRule {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	counts := make([]string, 0, len(ids))
	for _, id := range ids {
		counts = append(counts, fmt.Sprintf("%s=%d", id, report.CountsByRule[id]))
	}
	if _, err := fmt.Fprintf(w, "counts\t%s\n", strings.Join(counts, " ")); err != nil {
		return err
	}
	return w.Flush()
}

// lineColumn maps a UTF-8 byte offset to a 1-based line and rune column.
func lineColumn(text string, offset int) (int, int) {
	if offset > len(text) {
		offset = len(text)
	}
	prefix := text[:offset]
	line := strings.Count(prefix, "\n") + 1
	col := utf8.RuneCountInString(prefix[strings.LastIndexByte(prefix, '\n')+1:]) + 1
	return line, col
}

var compactEscaper = strings.NewReplacer(`\`, `\\`, "\n", `\n`, "\r", `\r`, "\t", `\t`)

func compactEscape(s string) string { return compactEscaper.Replace(s) }
