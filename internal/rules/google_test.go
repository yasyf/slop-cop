package rules

import (
	"testing"

	"github.com/yasyf/slop-cop/internal/types"
)

func idSet(rs []types.ViolationRule) map[string]struct{} {
	m := make(map[string]struct{}, len(rs))
	for _, r := range rs {
		m[r.ID] = struct{}{}
	}
	return m
}

// The google layer overrides by silencing, so every override has to name a
// rule that exists, and the keys have to be google rules doing the overriding.
func TestSupersedesReferencesRealRules(t *testing.T) {
	google := idSet(Google)
	for googleID, silenced := range Supersedes {
		if _, ok := google[googleID]; !ok {
			t.Errorf("Supersedes key %q is not a google rule", googleID)
		}
		for _, id := range silenced {
			if _, ok := ByID[id]; !ok {
				t.Errorf("Supersedes[%q] names unknown rule %q", googleID, id)
			}
		}
	}
}

// Prompt numbering is positional, so google has to sit after slop and base or
// every existing prompt golden shifts.
func TestAllAppendsGoogleLast(t *testing.T) {
	if want := len(Slop) + len(Base) + len(Google); len(All) != want {
		t.Fatalf("len(All) = %d, want %d", len(All), want)
	}
	for i, r := range Google {
		got := All[len(Slop)+len(Base)+i]
		if got.ID != r.ID {
			t.Errorf("All[%d] = %q, want google rule %q", len(Slop)+len(Base)+i, got.ID, r.ID)
		}
	}
}

func TestGoogleRulesAreWellFormed(t *testing.T) {
	for _, r := range Google {
		if r.Category != types.CategoryGoogle {
			t.Errorf("%s: category is %q, want %q", r.ID, r.Category, types.CategoryGoogle)
		}
		if r.Cite == "" {
			t.Errorf("%s: empty Cite; CC BY 4.0 attribution is per-rule", r.ID)
		}
		if r.Name == "" || r.Description == "" || r.Tip == "" {
			t.Errorf("%s: missing Name, Description, or Tip", r.ID)
		}
		if r.RequiresLLM && r.LLMTier == "" {
			t.Errorf("%s: RequiresLLM with no LLMTier", r.ID)
		}
		if !r.RequiresLLM && r.LLMTier != "" {
			t.Errorf("%s: client-side rule carries LLMTier %q", r.ID, r.LLMTier)
		}
	}
}

func TestSupersedesIsNotSelfReferential(t *testing.T) {
	for googleID, silenced := range Supersedes {
		for _, id := range silenced {
			if id == googleID {
				t.Errorf("Supersedes[%q] names itself", googleID)
			}
		}
	}
}
