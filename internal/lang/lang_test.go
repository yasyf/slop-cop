package lang_test

import (
	"testing"

	"github.com/yasyf/slop-cop/internal/lang"
	"github.com/yasyf/slop-cop/internal/types"
)

type fakeAnalyzer struct{ name string }

func (f *fakeAnalyzer) Name() string { return f.name }
func (f *fakeAnalyzer) Analyze(src string) (string, []lang.Range, error) {
	return src, nil, nil
}
func (f *fakeAnalyzer) ApplySuppressions(vs []types.Violation, _ []lang.Range, _ string) []types.Violation {
	return vs
}

func TestTextResolvesToNil(t *testing.T) {
	a, ok := lang.ByName("text")
	if !ok {
		t.Fatal(`ByName("text") reported unknown`)
	}
	if a != nil {
		t.Fatalf(`ByName("text") = %v, want nil`, a)
	}
}

func TestByNameUnknown(t *testing.T) {
	if _, ok := lang.ByName("definitely-not-a-language"); ok {
		t.Fatal("expected unknown name to report false")
	}
}

func TestOverlapsAndCount(t *testing.T) {
	spans := []lang.Range{
		{Start: 0, End: 10, Kind: lang.KindListItem},
		{Start: 12, End: 18, Kind: lang.KindListItem},
		{Start: 20, End: 30, Kind: lang.KindHeading},
	}
	if !lang.Overlaps(5, 15, spans, lang.KindListItem) {
		t.Fatal("Overlaps missed [5,15) vs list items")
	}
	if lang.Overlaps(30, 40, spans, lang.KindListItem) {
		t.Fatal("Overlaps false-positive on wrong kind")
	}
	if got := lang.CountOverlapping(5, 15, spans, lang.KindListItem); got != 2 {
		t.Fatalf("CountOverlapping [5,15) = %d, want 2", got)
	}
}

func TestRestoreMatchedText(t *testing.T) {
	v := types.Violation{StartIndex: 3, EndIndex: 8, MatchedText: "     "}
	lang.RestoreMatchedText(&v, "hello world")
	if v.MatchedText != "lo wo" {
		t.Fatalf("MatchedText = %q, want %q", v.MatchedText, "lo wo")
	}
	// Bad indices leave the value alone rather than panicking.
	v2 := types.Violation{StartIndex: -1, EndIndex: 4, MatchedText: "orig"}
	lang.RestoreMatchedText(&v2, "hello")
	if v2.MatchedText != "orig" {
		t.Fatalf("bad range mutated MatchedText: %q", v2.MatchedText)
	}
}
