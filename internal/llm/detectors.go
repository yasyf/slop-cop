package llm

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/yasyf/slop-cop/internal/types"
)

// Default models and timeouts match the TS source.
const (
	DefaultSentenceModel = "claude-haiku-4-5-20251001"
	DefaultDocumentModel = "claude-sonnet-4-6"

	DefaultSentenceTimeout = 30 * time.Second
	DefaultDocumentTimeout = 60 * time.Second

	chunkThreshold = 4000
	chunkSize      = 3500
)

// Options governs an LLM detection or rewrite invocation. Bin, extra args and
// model may be overridden; zero values pick TS-source defaults.
type Options struct {
	Bin       string
	Model     string
	Timeout   time.Duration
	ExtraArgs []string
}

func (o Options) config(defaultModel string, defaultTimeout time.Duration) Config {
	model := o.Model
	if model == "" {
		model = defaultModel
	}
	timeout := o.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}
	return Config{Bin: o.Bin, Model: model, Timeout: timeout, ExtraArgs: o.ExtraArgs}
}

type llmResult struct {
	RuleID          string `json:"ruleId"`
	MatchedText     string `json:"matchedText"`
	Explanation     string `json:"explanation"`
	SuggestedChange string `json:"suggestedChange"`
}

type violationsEnvelope struct {
	Violations []llmResult `json:"violations"`
}

// RunSentence is the fast-pass (Haiku) sentence-level detector. Long inputs
// are chunked on paragraph boundaries, analysed in parallel, and merged.
func RunSentence(ctx context.Context, text string, opts Options) ([]types.Violation, error) {
	cfg := opts.config(DefaultSentenceModel, DefaultSentenceTimeout)
	chunks := chunkText(text)
	if len(chunks) == 1 {
		return callDetector(ctx, cfg, SentenceSystemPrompt, BuildSentencePrompt(), text, text)
	}

	results := make([][]types.Violation, len(chunks))
	errs := make([]error, len(chunks))
	var wg sync.WaitGroup
	for i, chunk := range chunks {
		wg.Add(1)
		go func(i int, chunk string) {
			defer wg.Done()
			vs, err := callDetector(ctx, cfg, SentenceSystemPrompt, BuildSentencePrompt(), chunk, text)
			if err != nil {
				errs[i] = err
				return
			}
			results[i] = vs
		}(i, chunk)
	}
	wg.Wait()

	var merged []types.Violation
	for _, vs := range results {
		merged = append(merged, vs...)
	}
	// Propagate the first chunk error only if *every* chunk failed — a
	// single chunk missing is a degraded-but-useful result in the TS source.
	allFailed := true
	for _, err := range errs {
		if err == nil {
			allFailed = false
			break
		}
	}
	if allFailed && len(errs) > 0 {
		return nil, errs[0]
	}
	return dedupeByRuleAndMatch(merged), nil
}

// RunDocument is the deep-pass (Sonnet) document-level detector.
func RunDocument(ctx context.Context, text string, opts Options) ([]types.Violation, error) {
	cfg := opts.config(DefaultDocumentModel, DefaultDocumentTimeout)
	return callDetector(ctx, cfg, DocumentSystemPrompt, BuildDocumentPrompt(), text, text)
}

func callDetector(ctx context.Context, cfg Config, system, rulesPrompt, analyse, full string) ([]types.Violation, error) {
	user := rulesPrompt + "\n\nText to analyze:\n\n" + analyse
	var env violationsEnvelope
	if err := RunSchema(ctx, cfg, system, user, json.RawMessage(ViolationToolSchema), &env); err != nil {
		return nil, err
	}
	return processViolations(full, env.Violations), nil
}

// processViolations mirrors the TS helper: resolve each match to a byte
// offset in the full text (case-sensitive first, case-insensitive fallback),
// and sanitize suggestedChange.
func processViolations(text string, items []llmResult) []types.Violation {
	if len(items) == 0 {
		return nil
	}
	lower := strings.ToLower(text)
	out := make([]types.Violation, 0, len(items))
	for _, item := range items {
		if item.RuleID == "" || item.MatchedText == "" {
			continue
		}
		suggestion := sanitizeSuggestedChange(item.SuggestedChange, item.MatchedText)
		idx := strings.Index(text, item.MatchedText)
		if idx == -1 {
			fallback := strings.Index(lower, strings.ToLower(item.MatchedText))
			if fallback == -1 {
				continue
			}
			end := fallback + len(item.MatchedText)
			out = append(out, types.Violation{
				RuleID:          item.RuleID,
				StartIndex:      fallback,
				EndIndex:        end,
				MatchedText:     text[fallback:end],
				Explanation:     item.Explanation,
				SuggestedChange: suggestion,
			})
			continue
		}
		end := idx + len(item.MatchedText)
		out = append(out, types.Violation{
			RuleID:          item.RuleID,
			StartIndex:      idx,
			EndIndex:        end,
			MatchedText:     item.MatchedText,
			Explanation:     item.Explanation,
			SuggestedChange: suggestion,
		})
	}
	return out
}

// instructionPrefix detects action verbs signalling that the model wrote an
// instruction instead of a replacement. Ported from the TS INSTRUCTION_PREFIX.
var instructionPrefix = regexp.MustCompile(`(?i)^(remove|delete|cut|eliminate|omit|replace|rewrite|revise|change|consider|rephrase)\b`)

func sanitizeSuggestedChange(suggestion, matched string) string {
	if suggestion == "" {
		return suggestion
	}
	trimmed := strings.TrimSpace(suggestion)
	if instructionPrefix.MatchString(trimmed) && len(suggestion) > int(float64(len(matched))*1.5) {
		return ""
	}
	return suggestion
}

// chunkText ports the TS paragraph-boundary chunker, including the
// "overlap by one paragraph" behaviour at boundaries.
func chunkText(text string) []string {
	if len(text) <= chunkThreshold {
		return []string{text}
	}
	paragraphs := regexp.MustCompile(`\n\n+`).Split(text, -1)
	var chunks []string
	var current []string
	length := 0
	for i, para := range paragraphs {
		current = append(current, para)
		length += len(para) + 2
		if length >= chunkSize && i < len(paragraphs)-1 {
			chunks = append(chunks, strings.Join(current, "\n\n"))
			current = []string{para}
			length = len(para) + 2
		}
	}
	if len(current) > 0 {
		chunks = append(chunks, strings.Join(current, "\n\n"))
	}
	return chunks
}

func dedupeByRuleAndMatch(in []types.Violation) []types.Violation {
	seen := make(map[string]struct{}, len(in))
	out := make([]types.Violation, 0, len(in))
	for _, v := range in {
		key := v.RuleID + ":" + v.MatchedText
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, v)
	}
	return out
}
