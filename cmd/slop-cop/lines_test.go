package main

import (
	"testing"

	"github.com/yasyf/slop-cop/internal/types"
)

func TestParseLineRange(t *testing.T) {
	cases := []struct {
		raw         string
		wantPresent bool
		wantStart   int
		wantEnd     int
	}{
		{"", false, 0, 0},
		{"   ", false, 0, 0},
		{"50", true, 50, 50},
		{" 50 ", true, 50, 50},
		{"50:80", true, 50, 80},
		{"50 : 80", true, 50, 80},
		{"50:", true, 50, 0},
		{":80", true, 1, 80},
		{":", true, 1, 0},
		{"7:7", true, 7, 7},
	}
	for _, c := range cases {
		lr, present, err := parseLineRange(c.raw)
		if err != nil {
			t.Fatalf("parseLineRange(%q) unexpected err=%v", c.raw, err)
		}
		if present != c.wantPresent {
			t.Fatalf("parseLineRange(%q) present=%v, want %v", c.raw, present, c.wantPresent)
		}
		if present && (lr.start != c.wantStart || lr.end != c.wantEnd) {
			t.Fatalf("parseLineRange(%q) = {%d,%d}, want {%d,%d}", c.raw, lr.start, lr.end, c.wantStart, c.wantEnd)
		}
	}
}

func TestParseLineRangeErrors(t *testing.T) {
	for _, raw := range []string{"0", "-3", "abc", "10:5", "5:x", "x:5", "1:0"} {
		if _, _, err := parseLineRange(raw); err == nil {
			t.Fatalf("parseLineRange(%q) expected error, got nil", raw)
		}
	}
}

func TestLineStartByte(t *testing.T) {
	text := "a\nbb\nccc" // line1@0 "a", line2@2 "bb", line3@5 "ccc"
	cases := []struct {
		n    int
		want int
	}{
		{0, 0}, {1, 0}, {2, 2}, {3, 5}, {4, len(text)}, {99, len(text)},
	}
	for _, c := range cases {
		if got := lineStartByte(text, c.n); got != c.want {
			t.Fatalf("lineStartByte(%q, %d) = %d, want %d", text, c.n, got, c.want)
		}
	}
}

func TestLineStartByteTrailingNewline(t *testing.T) {
	text := "a\nb\n" // line1@0, line2@2, line3 (empty) @4 == len
	if got := lineStartByte(text, 3); got != len(text) {
		t.Fatalf("lineStartByte trailing newline = %d, want %d", got, len(text))
	}
}

func TestLineByteSpan(t *testing.T) {
	text := "a\nbb\nccc"
	cases := []struct {
		lr             lineRange
		wantLo, wantHi int
	}{
		{lineRange{1, 1}, 0, 2},                  // "a\n"
		{lineRange{2, 2}, 2, 5},                  // "bb\n"
		{lineRange{2, 3}, 2, len(text)},          // "bb\nccc"
		{lineRange{2, 0}, 2, len(text)},          // open end
		{lineRange{1, 0}, 0, len(text)},          // whole file
		{lineRange{99, 0}, len(text), len(text)}, // start past EOF → empty
	}
	for _, c := range cases {
		lo, hi := lineByteSpan(text, c.lr)
		if lo != c.wantLo || hi != c.wantHi {
			t.Fatalf("lineByteSpan(%q, %+v) = [%d,%d), want [%d,%d)", text, c.lr, lo, hi, c.wantLo, c.wantHi)
		}
	}
}

func TestFilterByLines(t *testing.T) {
	// Three lines. "docSpan" starts on line 1 but runs to EOF (a document-level
	// tell); "span12" starts on line 1 and crosses into line 2. Both are
	// attributed to line 1 because that's where they begin.
	text := "aaaa\nbbbb\ncccc"
	vs := []types.Violation{
		{RuleID: "line1", StartIndex: 0, EndIndex: 4},    // begins line 1
		{RuleID: "span12", StartIndex: 2, EndIndex: 7},   // begins line 1, crosses into line 2
		{RuleID: "docSpan", StartIndex: 0, EndIndex: 14}, // begins line 1, spans whole doc
		{RuleID: "line2", StartIndex: 5, EndIndex: 9},    // begins line 2
		{RuleID: "line3", StartIndex: 10, EndIndex: 14},  // begins line 3
	}
	has := func(got []types.Violation, id string) bool {
		for _, v := range got {
			if v.RuleID == id {
				return true
			}
		}
		return false
	}

	// Line 2 only: just the hit that begins on line 2. The line-1-anchored
	// spans (span12, docSpan) do NOT leak in.
	got := filterByLines(vs, text, lineRange{2, 2})
	if len(got) != 1 || !has(got, "line2") {
		t.Fatalf("lines 2:2 = %+v, want line2 only", got)
	}

	// Line 1 collects every violation that begins there.
	got = filterByLines(vs, text, lineRange{1, 1})
	if len(got) != 3 || !has(got, "line1") || !has(got, "span12") || !has(got, "docSpan") {
		t.Fatalf("lines 1:1 = %+v, want line1 + span12 + docSpan", got)
	}

	// Open-ended from line 2 keeps only hits beginning on lines 2-3.
	got = filterByLines(vs, text, lineRange{2, 0})
	if len(got) != 2 || !has(got, "line2") || !has(got, "line3") {
		t.Fatalf("lines 2: = %+v, want line2 + line3", got)
	}

	// A start past EOF matches nothing.
	if got = filterByLines(vs, text, lineRange{99, 0}); len(got) != 0 {
		t.Fatalf("lines 99: = %+v, want empty", got)
	}
}
