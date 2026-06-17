package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/yasyf/slop-cop/internal/types"
)

// lineRange is a 1-based inclusive span of input lines. An end of 0 means
// "open-ended" — through the last line; mapping to bytes clamps it to EOF.
type lineRange struct {
	start int
	end   int // 0 == open (to EOF)
}

// parseLineRange parses the --lines flag. Accepted forms (1-based, inclusive):
//
//	"50"     a single line       → {50, 50}
//	"50:80"  a closed range       → {50, 80}
//	"50:"    open-ended to EOF    → {50, 0}
//	":80"    from the first line  → {1, 80}
//
// The second result is false when raw is empty (no range requested). A
// malformed value returns an error for the caller to surface as a usage error.
func parseLineRange(raw string) (lineRange, bool, error) {
	if strings.TrimSpace(raw) == "" {
		return lineRange{}, false, nil
	}
	parse := func(s string) (int, error) {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return 0, fmt.Errorf("invalid --lines %q: %q is not a line number", raw, strings.TrimSpace(s))
		}
		if n < 1 {
			return 0, fmt.Errorf("invalid --lines %q: line numbers are 1-based", raw)
		}
		return n, nil
	}

	lo, hi, hasColon := strings.Cut(raw, ":")
	if !hasColon {
		n, err := parse(lo)
		if err != nil {
			return lineRange{}, false, err
		}
		return lineRange{start: n, end: n}, true, nil
	}

	lr := lineRange{start: 1, end: 0}
	if strings.TrimSpace(lo) != "" {
		n, err := parse(lo)
		if err != nil {
			return lineRange{}, false, err
		}
		lr.start = n
	}
	if strings.TrimSpace(hi) != "" {
		n, err := parse(hi)
		if err != nil {
			return lineRange{}, false, err
		}
		lr.end = n
	}
	if lr.end != 0 && lr.end < lr.start {
		return lineRange{}, false, fmt.Errorf("invalid --lines %q: end line %d precedes start line %d", raw, lr.end, lr.start)
	}
	return lr, true, nil
}

// filterByLines keeps only violations that begin within the requested line
// range. Detectors always run over the full input — so context-sensitive rules
// and the LLM passes see the whole document — and this trims the report to the
// lines the caller cares about (e.g. the lines an edit just touched). A
// violation is attributed to the line its span starts on, so a document-level
// span (one that begins at the top of the file, like a staccato burst) doesn't
// surface on every range.
func filterByLines(vs []types.Violation, text string, lr lineRange) []types.Violation {
	startByte, endByte := lineByteSpan(text, lr)
	var out []types.Violation
	for _, v := range vs {
		if v.StartIndex >= startByte && v.StartIndex < endByte {
			out = append(out, v)
		}
	}
	return out
}

// lineByteSpan maps a 1-based inclusive line range to a [start, end) byte span
// in text. An open end (lr.end == 0) or an end past EOF extends to len(text);
// a start past EOF yields an empty span so nothing matches.
func lineByteSpan(text string, lr lineRange) (int, int) {
	startByte := lineStartByte(text, lr.start)
	if lr.end == 0 {
		return startByte, len(text)
	}
	// The end of line N is the start of line N+1 (or EOF).
	return startByte, lineStartByte(text, lr.end+1)
}

// lineStartByte returns the byte offset where 1-based line n begins. For n past
// the last line it returns len(text); for n <= 1 it returns 0.
func lineStartByte(text string, n int) int {
	if n <= 1 {
		return 0
	}
	seen := 1
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			if seen++; seen == n {
				return i + 1
			}
		}
	}
	return len(text)
}
