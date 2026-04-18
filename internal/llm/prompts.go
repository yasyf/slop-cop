package llm

import (
	"fmt"
	"strings"

	"github.com/yasyf/slop-cop/internal/rules"
	"github.com/yasyf/slop-cop/internal/types"
)

// Prompts ported 1:1 from src/detectors/llmDetectors.ts.

const SentenceSystemPrompt = `You are an expert editor analyzing text for LLM-generated prose patterns.
You will be given a passage and asked to identify specific rhetorical and structural tells.
Be conservative — only flag clear, unambiguous instances.`

const DocumentSystemPrompt = `You are an experienced editor reading a complete piece of writing to identify structural and compositional problems that only become visible at document scale — patterns that emerge across paragraphs rather than within a single sentence.
Be conservative — only flag clear, unambiguous cases.`

// BuildSentencePrompt mirrors buildLLMRulesPrompt() in the TS source.
func BuildSentencePrompt() string {
	return buildRulePrompt(
		types.LLMTierSentence,
		"Identify these patterns:",
		"For suggestedChange: rewrite only the matched span. Make it direct and concrete.",
	)
}

// BuildDocumentPrompt mirrors buildDocumentRulesPrompt().
func BuildDocumentPrompt() string {
	return buildRulePrompt(
		types.LLMTierDocument,
		"Read the entire piece as an editor. Identify these document-level patterns:",
		"Return only clear cases. If the piece is short, tight, or well-structured, return [].",
	)
}

func buildRulePrompt(tier types.LLMTier, header, footer string) string {
	var parts []string
	i := 1
	for _, r := range rules.All {
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
