package llm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yasyf/slop-cop/internal/rules"
	"github.com/yasyf/slop-cop/internal/types"
)

// TestPromptsCarryTheirTier checks each LLM-tier rule appears in exactly the
// prompt for its own tier.
func TestPromptsCarryTheirTier(t *testing.T) {
	sentence := BuildSentencePrompt(rules.All)
	document := BuildDocumentPrompt(rules.All)

	sentenceRules, documentRules := 0, 0
	for _, r := range rules.All {
		needle := `"` + r.ID + `":`
		switch r.LLMTier {
		case types.LLMTierSentence:
			sentenceRules++
			if !strings.Contains(sentence, needle) {
				t.Errorf("sentence prompt missing %s", r.ID)
			}
			if strings.Contains(document, needle) {
				t.Errorf("document prompt wrongly includes %s", r.ID)
			}
		case types.LLMTierDocument:
			documentRules++
			if !strings.Contains(document, needle) {
				t.Errorf("document prompt missing %s", r.ID)
			}
			if strings.Contains(sentence, needle) {
				t.Errorf("sentence prompt wrongly includes %s", r.ID)
			}
		}
	}
	if sentenceRules == 0 || documentRules == 0 {
		t.Fatalf("rule catalogue has %d sentence-tier and %d document-tier rules, want both > 0", sentenceRules, documentRules)
	}
}

// readGolden drops the single trailing newline the repo's end-of-file-fixer
// pre-commit hook adds to every file. A prompt never ends in a newline, so the
// comparison stays exact on the bytes that matter.
func readGolden(t *testing.T, name string) string {
	t.Helper()
	//nolint:gosec // G304: a fixed testdata path, not user input.
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSuffix(string(b), "\n")
}

// TestSlopPromptsMatchPreBaseLayerGoldens is the prompt half of the layer
// equivalence gate. The goldens were captured by running the pre-base-layer
// BuildSentencePrompt/BuildDocumentPrompt — which read rules.All when All was
// the 48 slop rules — so a byte match proves --standard=slop asks the model
// exactly what slop-cop asked before the base layer existed. Regenerating
// these files instead of fixing the drift defeats the gate.
func TestSlopPromptsMatchPreBaseLayerGoldens(t *testing.T) {
	cases := []struct {
		golden string
		got    string
	}{
		{"sentence_prompt_slop.golden", BuildSentencePrompt(rules.Slop)},
		{"document_prompt_slop.golden", BuildDocumentPrompt(rules.Slop)},
	}
	for _, c := range cases {
		t.Run(c.golden, func(t *testing.T) {
			want := readGolden(t, c.golden)
			if c.got == want {
				return
			}
			t.Fatalf("prompt drifted from the pre-base-layer golden %s\n%s", c.golden, firstDiff(want, c.got))
		})
	}
}

// TestAllPromptAppendsLaterLayersAfterSlop pins the numbering contract: the
// combined catalogue reproduces the slop-only prompt's numbered entries
// verbatim and in order, then continues, so no existing rule's number moves.
func TestAllPromptAppendsLaterLayersAfterSlop(t *testing.T) {
	cases := []struct {
		tier types.LLMTier
		slop string
		all  string
	}{
		{types.LLMTierSentence, BuildSentencePrompt(rules.Slop), BuildSentencePrompt(rules.All)},
		{types.LLMTierDocument, BuildDocumentPrompt(rules.Slop), BuildDocumentPrompt(rules.All)},
	}
	for _, c := range cases {
		t.Run(string(c.tier), func(t *testing.T) {
			slopEntries := promptEntries(c.slop)
			allEntries := promptEntries(c.all)
			if len(allEntries) <= len(slopEntries) {
				t.Fatalf("all-layer prompt has %d entries, slop-only has %d; base rules did not append", len(allEntries), len(slopEntries))
			}
			for i, want := range slopEntries {
				if allEntries[i] != want {
					t.Fatalf("entry %d moved:\n slop: %q\n  all: %q", i+1, want, allEntries[i])
				}
			}
			for _, entry := range allEntries[len(slopEntries):] {
				if !ruleIsBase(entry) && !ruleIsGoogle(entry) {
					t.Fatalf("entry appended after the slop block is neither a base nor a google rule: %q", entry)
				}
			}
		})
	}
}

// TestBasePromptCarriesOnlyBaseRules proves the filter is real: no slop rule
// reaches a --standard=base prompt.
func TestBasePromptCarriesOnlyBaseRules(t *testing.T) {
	for _, prompt := range []string{BuildSentencePrompt(rules.Base), BuildDocumentPrompt(rules.Base)} {
		entries := promptEntries(prompt)
		if len(entries) != 1 {
			t.Fatalf("base-only prompt has %d entries, want 1: %q", len(entries), entries)
		}
		if !strings.HasPrefix(entries[0], `1. "`) {
			t.Fatalf("base-only prompt does not renumber from 1: %q", entries[0])
		}
		if !ruleIsBase(entries[0]) {
			t.Fatalf("base-only prompt carries a non-base rule: %q", entries[0])
		}
	}
}

// TestOptionsRulesReachTheProcess closes the loop between the layer selection
// and the subprocess: a rule the caller filters out must never reach the
// prompt claude is fed, not merely be dropped from the response. spawnllm
// delivers the prompt on stdin, so that is what the stand-in captures.
func TestOptionsRulesReachTheProcess(t *testing.T) {
	fixture, err := filepath.Abs(filepath.Join("testdata", "claude_stream_array.json"))
	if err != nil {
		t.Fatal(err)
	}
	capture := filepath.Join(t.TempDir(), "stdin")
	dir := t.TempDir()
	script := fmt.Sprintf("#!/bin/sh\n/bin/cat > %q\nexec /bin/cat %q\n", capture, fixture)
	//nolint:gosec // G306: an executable stub on a test-owned temp PATH.
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	if _, err := RunSentence(context.Background(), "Some prose to analyse.", Options{
		Timeout: 30 * time.Second,
		Rules:   rules.Base,
	}); err != nil {
		t.Fatalf("RunSentence: %v", err)
	}

	//nolint:gosec // G304: a test-owned temp path, not user input.
	b, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	prompt := string(b)
	if !strings.Contains(prompt, `"agentless-passive":`) {
		t.Fatalf("the base sentence rule never reached claude:\n%s", prompt)
	}
	for _, r := range rules.Slop {
		if r.LLMTier != types.LLMTierSentence {
			continue
		}
		if strings.Contains(prompt, `"`+r.ID+`":`) {
			t.Fatalf("slop rule %s reached claude despite Rules=rules.Base", r.ID)
		}
	}
}

// promptEntries returns the numbered rule blocks of a prompt, dropping the
// header and footer buildRulePrompt wraps them in.
func promptEntries(prompt string) []string {
	parts := strings.Split(prompt, "\n\n")
	return parts[1 : len(parts)-1]
}

// ruleIsBase reports whether a numbered prompt entry names a base-layer rule.
func ruleIsGoogle(entry string) bool {
	for _, r := range rules.Google {
		if strings.Contains(entry, `"`+r.ID+`":`) {
			return true
		}
	}
	return false
}

func ruleIsBase(entry string) bool {
	for _, r := range rules.Base {
		if strings.Contains(entry, `"`+r.ID+`":`) {
			return true
		}
	}
	return false
}

// firstDiff renders the first differing line of two strings, so a drifted
// prompt reports the change rather than 3KB of context.
func firstDiff(want, got string) string {
	wantLines := strings.Split(want, "\n")
	gotLines := strings.Split(got, "\n")
	for i := 0; i < len(wantLines) && i < len(gotLines); i++ {
		if wantLines[i] != gotLines[i] {
			return fmt.Sprintf("line %d:\n want: %q\n  got: %q", i+1, wantLines[i], gotLines[i])
		}
	}
	return fmt.Sprintf("line counts differ: want %d, got %d", len(wantLines), len(gotLines))
}
