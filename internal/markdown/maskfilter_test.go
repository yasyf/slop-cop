package markdown

import (
	"testing"

	"github.com/yasyf/slop-cop/internal/detectors"
	"github.com/yasyf/slop-cop/internal/types"
)

// reportedRules runs the full deterministic pipeline the CLI runs — analyze,
// every detector layer over the masked copy, then the post filter — and
// returns the rule IDs that survive.
func reportedRules(t *testing.T, src string) map[string]bool {
	t.Helper()
	masked, suppress, _ := Analyze(src)
	vs := detectors.RunClient(masked)
	vs = append(vs, detectors.RunBase(masked)...)
	vs = append(vs, detectors.RunGoogle(masked)...)
	out := map[string]bool{}
	for _, v := range ApplySuppressions(detectors.Deduplicate(vs), suppress, src) {
		out[v.RuleID] = true
	}
	return out
}

// TestMaskedFillerNeverSatisfiesALengthQuantifier pins both directions of the
// masking fix: a sentence whose only "elaboration" or "qualifier" is an inline
// code span reports nothing, while the same sentence carrying real prose in
// that position still reports.
func TestMaskedFillerNeverSatisfiesALengthQuantifier(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		want    string
		unwant  string
		wantHit bool
	}{
		{
			name:   "colon elaborated by a code span",
			src:    "Resolve the lane: `ccx vcs info --json --refresh --no-gt` prints it.\n",
			unwant: "colon-elaboration",
		},
		{
			name:    "colon elaborated by prose",
			src:     "Resolve the lane by hand: the reconciliation pass rewrites it for you.\n",
			want:    "colon-elaboration",
			wantHit: true,
		},
		{
			name:   "parenthetical holding only a code span",
			src:    "The lookup is layer-agnostic (`internal/lang/suppress_data.go:6-176`) and works today.\n",
			unwant: "parenthetical-qualifier",
		},
		{
			name:    "parenthetical holding prose",
			src:     "The lookup is layer-agnostic (it never inspects the layer) and works today.\n",
			want:    "parenthetical-qualifier",
			wantHit: true,
		},
		{
			name:   "parenthetical holding a code span plus too little prose",
			src:    "The lookup is layer-agnostic (the `--refresh` flag) and works today.\n",
			unwant: "parenthetical-qualifier",
		},
		{
			name:    "parenthetical holding a code span plus enough prose",
			src:     "The lookup is layer-agnostic (it never inspects the `--refresh` flag) and works today.\n",
			want:    "parenthetical-qualifier",
			wantHit: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := reportedRules(t, c.src)
			if c.wantHit && !got[c.want] {
				t.Fatalf("%s should still fire; got %v", c.want, got)
			}
			if !c.wantHit && got[c.unwant] {
				t.Fatalf("%s fired on masking filler; got %v", c.unwant, got)
			}
		})
	}
}

// TestGFMTableCellsAreStructural pins the table extension: a pipe-less GFM
// table is a table to the parser, so its cells carry KindTableCell and the
// sentence-grammar rules that read a row as a runaway sentence stay quiet.
// The same words as a paragraph still report.
func TestGFMTableCellsAreStructural(t *testing.T) {
	table := "The runner reports one verdict for each stage of the pipeline.\n\n" +
		"Flag | What it does\n--- | ---\n" +
		"llm | The runner starts the build, runs the tests and ships the artifact\n" +
		"deep | It counts every single row that the parser has found so far\n"
	_, suppress, _ := Analyze(table)
	cells := 0
	for _, r := range suppress {
		if r.Kind == KindTableCell {
			cells++
		}
	}
	if cells != 6 {
		t.Fatalf("table-cell ranges = %d, want 6: %+v", cells, suppress)
	}
	if got := reportedRules(t, table); got["missing-terminal-period"] {
		t.Fatalf("missing-terminal-period fired on a table row; got %v", got)
	}

	paragraph := "The runner reports one verdict for each stage of the pipeline.\n\n" +
		"It counts every single row that the parser has found so far\n"
	if got := reportedRules(t, paragraph); !got["missing-terminal-period"] {
		t.Fatalf("missing-terminal-period should fire on the same words as prose; got %v", got)
	}
}

// TestAnalyzeReportsWhatItMasked asserts every masked byte is covered by a
// KindMasked range, the input DropMaskMatches reads to tell filler from prose.
func TestAnalyzeReportsWhatItMasked(t *testing.T) {
	src := "---\ntitle: Notes\n---\n\nRun `slop-cop check -` on [the docs](https://example.com/docs) first.\n\n```go\nfmt.Println()\n```\n"
	masked, suppress, _ := Analyze(src)
	covered := make([]bool, len(src))
	for _, r := range suppress {
		if r.Kind != KindMasked {
			continue
		}
		for i := r.Start; i < r.End; i++ {
			covered[i] = true
		}
	}
	for i := 0; i < len(src); i++ {
		if masked[i] != src[i] && !covered[i] {
			t.Fatalf("byte %d masked but no KindMasked range covers it: %q", i, src[i:i+1])
		}
	}
}

// TestDropMaskMatchesLeavesLLMRulesAlone asserts a rule no detector can
// reproduce survives the filter: the re-run proves nothing about it.
func TestDropMaskMatchesLeavesLLMRulesAlone(t *testing.T) {
	src := "The lookup is layer-agnostic (`internal/lang/suppress_data.go`) and works today.\n"
	_, suppress, _ := Analyze(src)
	vs := []types.Violation{{RuleID: "throat-clearing", StartIndex: 0, EndIndex: len(src) - 1}}
	if out := ApplySuppressions(vs, suppress, src); len(out) != 1 {
		t.Fatalf("LLM-only rule should survive the mask filter; got %+v", out)
	}
}

// TestSlopRulesAreSuppressedOnStructure pins the structural entries the
// suppress table now carries for the slop layer: a colon or an em dash that
// belongs to a heading or a list marker is punctuation, not a rhetorical
// pivot, while the same construction in a paragraph still reports.
func TestSlopRulesAreSuppressedOnStructure(t *testing.T) {
	structural := "## The rule is plain: it drops matches the mask alone produced\n\n" +
		"- The rule is plain: it drops any match that only survived the mask.\n" +
		"- A bullet runs on — and then pivots — without earning either dash.\n"
	got := reportedRules(t, structural)
	for _, id := range []string{"colon-elaboration", "em-dash-pivot"} {
		if got[id] {
			t.Fatalf("%s fired on heading/list-item structure; got %v", id, got)
		}
	}

	prose := "The rule is plain: it drops any match that only survived the mask.\n\n" +
		"A paragraph runs on — and then pivots — without earning either dash.\n"
	got = reportedRules(t, prose)
	for _, id := range []string{"colon-elaboration", "em-dash-pivot"} {
		if !got[id] {
			t.Fatalf("%s should still fire in a paragraph; got %v", id, got)
		}
	}
}

// TestTableShapeIsNotProseRhythm pins the cell-boundary suppressions: a hit
// that straddles two or more table cells is the table's shape, while the same
// words as a paragraph — and a hit that sits inside one cell — still report.
func TestTableShapeIsNotProseRhythm(t *testing.T) {
	cases := []struct {
		name   string
		table  string
		prose  string
		ruleID string
	}{
		{
			name: "terse rows read as a staccato burst",
			table: "| Flag | Effect |\n| --- | --- |\n| off | Nothing runs. |\n" +
				"| low | One tier runs. |\n| high | Both tiers run. |\n| auto | It picks one. |\n",
			prose:  "Nothing runs. One tier runs. Both tiers run. It picks one.\n",
			ruleID: "staccato-burst",
		},
		{
			name:   "adjacent cells read as one phrase",
			table:  "| Flag | A | B |\n| --- | --- | --- |\n| off | none | none |\n",
			prose:  "The failure is not uncommon.\n",
			ruleID: "double-negative",
		},
		{
			name: "a pipe-less table reads as one long paragraph",
			table: "Flag | Effect\n--- | ---\na | It runs.\nb | It stops.\nc | It waits.\n" +
				"d | It retries.\ne | It reports.\nf | It exits.\ng | It logs.\n",
			prose: "It runs. It stops. It waits. It retries. It reports. It exits. It logs.\n",
			// Only long-paragraph is asserted; the prose form trips staccato-burst too.
			ruleID: "long-paragraph",
		},
		{
			name: "a pipe-less table reads as one long sentence",
			table: "Flag | Effect\n--- | ---\na | the runner starts the build and then runs the " +
				"tests and after that it ships the artifact while the report lands last and " +
				"none of this is a list and none of it is a heading either so it keeps going\n",
			prose: "The runner starts the build and then runs the tests and after that it ships " +
				"the artifact while the report lands last and none of this is a list and none " +
				"of it is a heading either so it keeps going.\n",
			ruleID: "long-sentence",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := reportedRules(t, c.table); got[c.ruleID] {
				t.Fatalf("%s fired across table cells; got %v", c.ruleID, got)
			}
			if got := reportedRules(t, c.prose); !got[c.ruleID] {
				t.Fatalf("%s should still fire on the same words as prose; got %v", c.ruleID, got)
			}
		})
	}
}

// TestDoubleNegativeInsideOneCellStillReports is the narrowness proof for the
// cell-boundary test: a blanket table-cell exclusion would have taken this hit
// with it, which is why the rule is straddling, not overlapping.
func TestDoubleNegativeInsideOneCellStillReports(t *testing.T) {
	src := "| Flag | Effect |\n| --- | --- |\n| a | It is not uncommon. |\n"
	if got := reportedRules(t, src); !got["double-negative"] {
		t.Fatalf("a real double negative inside one cell should report; got %v", got)
	}
}
