package llm

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/yasyf/slop-cop/internal/rules"
)

// Default knobs for the rewrite call. Ported from rewriteParagraph() in
// src/detectors/llmDetectors.ts (model claude-haiku-4-5-20251001, 20s).
const (
	DefaultRewriteModel   = "claude-haiku-4-5-20251001"
	DefaultRewriteTimeout = 20 * time.Second
)

// RewriteDefaultRuleIDs are always included as directives in the rewrite
// system prompt, regardless of which rules fired. Matches REWRITE_DEFAULT_RULE_IDS.
var RewriteDefaultRuleIDs = []string{
	"elevated-register",
	"filler-adverbs",
	"hedge-stack",
	"unnecessary-elaboration",
	"grandiose-stakes",
	"triple-construction",
	"em-dash-pivot",
	"balanced-take",
}

// rewriteMetaPrinciples match REWRITE_META_PRINCIPLES in the TS source.
var rewriteMetaPrinciples = []string{
	"- Write directly. Cut preamble and throat-clearing.",
	"- Don't add explanations or transitions the original didn't have.",
	"- Preserve the paragraph's factual content and core meaning exactly.",
}

// BuildRewriteSystemPrompt mirrors buildRewriteSystemPrompt() in the TS.
func BuildRewriteSystemPrompt(violatedRuleHints []string) string {
	var principles []string
	for _, id := range RewriteDefaultRuleIDs {
		if r, ok := rules.ByID[id]; ok && r.LLMDirective != "" {
			principles = append(principles, "- "+r.LLMDirective)
		}
	}
	principles = append(principles, rewriteMetaPrinciples...)
	base := "You are an expert editor. Rewrite the given text to read like natural, direct human prose. Apply all of these principles:\n" + strings.Join(principles, "\n")
	if len(violatedRuleHints) == 0 {
		return base
	}
	var hints []string
	for _, h := range violatedRuleHints {
		hints = append(hints, "- "+h)
	}
	return base + "\n\nThis text has specific problems to fix:\n" + strings.Join(hints, "\n")
}

type rewriteEnvelope struct {
	Rewritten string `json:"rewritten"`
}

// RewriteParagraph sends paragraph through claude with the rewrite system
// prompt, returning the rewritten text with outer whitespace trimmed.
func RewriteParagraph(ctx context.Context, paragraph string, violatedRuleHints []string, opts Options) (string, error) {
	cfg := opts.config(DefaultRewriteModel, DefaultRewriteTimeout)
	var env rewriteEnvelope
	if err := RunSchema(ctx, cfg, BuildRewriteSystemPrompt(violatedRuleHints), paragraph, json.RawMessage(RewriteSchema), &env); err != nil {
		return "", err
	}
	return strings.TrimSpace(env.Rewritten), nil
}
