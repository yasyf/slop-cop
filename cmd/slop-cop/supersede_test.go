package main

import (
	"testing"

	"github.com/yasyf/slop-cop/internal/rules"
	"github.com/yasyf/slop-cop/internal/types"
)

// The style guide wins the instance it covers, not the rule. A tell silenced
// on one sentence has to stay live on the next one.
func TestDropSupersededSilencesOnlyTheOverlappingSpan(t *testing.T) {
	prev := rules.Supersedes
	rules.Supersedes = map[string][]string{"colon-intro-fragment": {"colon-elaboration"}}
	t.Cleanup(func() { rules.Supersedes = prev })

	vs := []types.Violation{
		{RuleID: "colon-intro-fragment", StartIndex: 10, EndIndex: 30, MatchedText: "Run this command:"},
		{RuleID: "colon-elaboration", StartIndex: 20, EndIndex: 26, MatchedText: "mmand:"},
		{RuleID: "colon-elaboration", StartIndex: 80, EndIndex: 95, MatchedText: "The result: chaos"},
	}

	got := dropSuperseded(vs)
	if len(got) != 2 {
		t.Fatalf("kept %d violations, want 2: %+v", len(got), got)
	}
	for _, v := range got {
		if v.RuleID == "colon-elaboration" && v.StartIndex == 20 {
			t.Error("overlapping colon-elaboration survived supersession")
		}
	}
	if got[1].RuleID != "colon-elaboration" || got[1].StartIndex != 80 {
		t.Errorf("non-overlapping colon-elaboration was silenced: %+v", got)
	}
}

func TestDropSupersededIsAPassthroughWithoutOverrides(t *testing.T) {
	prev := rules.Supersedes
	rules.Supersedes = map[string][]string{}
	t.Cleanup(func() { rules.Supersedes = prev })

	vs := []types.Violation{{RuleID: "colon-elaboration", StartIndex: 1, EndIndex: 2}}
	if got := dropSuperseded(vs); len(got) != 1 {
		t.Fatalf("kept %d violations, want 1", len(got))
	}
}
