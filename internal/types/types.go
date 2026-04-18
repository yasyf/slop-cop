// Package types defines the shared types for slop-cop detection.
//
// Violations carry byte offsets into the UTF-8 input, not UTF-16 code units
// as in the original JavaScript port. Consumers doing substring operations
// with [Violation.StartIndex]/[Violation.EndIndex] should slice the input as
// bytes (Go's default) rather than runes.
package types

// ViolationCategory groups related rules.
type ViolationCategory string

const (
	CategoryWordChoice        ViolationCategory = "word-choice"
	CategorySentenceStructure ViolationCategory = "sentence-structure"
	CategoryRhetorical        ViolationCategory = "rhetorical"
	CategoryStructural        ViolationCategory = "structural"
	CategoryFraming           ViolationCategory = "framing"
)

// LLMTier identifies which LLM pass produces a rule's detections.
type LLMTier string

const (
	LLMTierSentence LLMTier = "sentence"
	LLMTierDocument LLMTier = "document"
)

// ViolationRule describes a single detectable pattern.
type ViolationRule struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Category         ViolationCategory `json:"category"`
	Description      string            `json:"description"`
	Tip              string            `json:"tip"`
	CanRemove        bool              `json:"canRemove"`
	Color            string            `json:"color"`
	BgColor          string            `json:"bgColor"`
	RequiresLLM      bool              `json:"requiresLLM"`
	LLMTier          LLMTier           `json:"llmTier,omitempty"`
	LLMDetectionHint string            `json:"llmDetectionHint,omitempty"`
	RewriteHint      string            `json:"rewriteHint,omitempty"`
	LLMDirective     string            `json:"llmDirective,omitempty"`
}

// Violation is a single flagged span in a piece of text.
type Violation struct {
	RuleID          string `json:"ruleId"`
	StartIndex      int    `json:"startIndex"`
	EndIndex        int    `json:"endIndex"`
	MatchedText     string `json:"matchedText"`
	Explanation     string `json:"explanation,omitempty"`
	SuggestedChange string `json:"suggestedChange,omitempty"`
}
