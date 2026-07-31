package llm

import (
	"fmt"
	"strings"

	"github.com/yasyf/slop-cop/internal/types"
)

// Prompts ported 1:1 from src/detectors/llmDetectors.ts.

// SentenceSystemPrompt is the system prompt for the per-sentence LLM pass.
const SentenceSystemPrompt = `You are an expert editor analyzing text for LLM-generated prose patterns.
You will be given a passage and asked to identify specific rhetorical and structural tells.
Be conservative — only flag clear, unambiguous instances.`

// DocumentSystemPrompt is the system prompt for the whole-document LLM pass.
const DocumentSystemPrompt = `You are an experienced editor reading a complete piece of writing to identify structural and compositional problems that only become visible at document scale — patterns that emerge across paragraphs rather than within a single sentence.
Be conservative — only flag clear, unambiguous cases.`

// BuildSentencePrompt mirrors buildLLMRulesPrompt() in the TS source. The
// caller supplies the catalogue, so a run restricted to one layer never puts
// the other layer's rules in front of the model.
func BuildSentencePrompt(catalogue []types.ViolationRule) string {
	return buildRulePrompt(
		catalogue,
		types.LLMTierSentence,
		"Identify these patterns:",
		"For suggestedChange: rewrite only the matched span. Make it direct and concrete.",
	)
}

// BuildDocumentPrompt mirrors buildDocumentRulesPrompt().
func BuildDocumentPrompt(catalogue []types.ViolationRule) string {
	return buildRulePrompt(
		catalogue,
		types.LLMTierDocument,
		"Read the entire piece as an editor. Identify these document-level patterns:",
		"Return only clear cases. If the piece is short, tight, or well-structured, return [].",
	)
}

// buildRulePrompt numbers the tier's rules 1..N in catalogue order, so a
// catalogue whose head is unchanged produces an unchanged numbering.
func buildRulePrompt(catalogue []types.ViolationRule, tier types.LLMTier, header, footer string) string {
	var parts []string
	i := 1
	for _, r := range catalogue {
		if r.LLMTier != tier {
			continue
		}
		hint := r.LLMDetectionHint
		if hint == "" {
			hint = r.Description
		}
		parts = append(parts, fmt.Sprintf(`%d. "%s": %s`, i, r.ID, hint))
		i++
	}
	return header + "\n\n" + strings.Join(parts, "\n\n") + "\n\n" + footer
}

// ViolationToolSchema is the JSON schema passed to claude via --json-schema
// for the detection calls. It mirrors VIOLATION_TOOL_SCHEMA in the TS source.
const ViolationToolSchema = `{
  "type": "object",
  "properties": {
    "violations": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "ruleId":          {"type": "string"},
          "matchedText":     {"type": "string"},
          "explanation":     {"type": "string"},
          "suggestedChange": {"type": "string"}
        },
        "required": ["ruleId", "matchedText", "explanation", "suggestedChange"]
      }
    }
  },
  "required": ["violations"]
}`

// RewriteSchema is the JSON schema for rewriteParagraph responses.
const RewriteSchema = `{
  "type": "object",
  "properties": {
    "rewritten": {"type": "string"}
  },
  "required": ["rewritten"]
}`
