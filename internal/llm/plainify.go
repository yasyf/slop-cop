package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	spawnllm "github.com/yasyf/spawnllm/go"
)

// plainifyConcurrency caps the claude processes one batch runs at once. A
// register of findings arrives as a single array and each entry is its own
// call, so the fan-out is the caller's array length rather than a document's
// chunk count.
const plainifyConcurrency = 4

// PlainifySchema is the JSON schema for plainify responses.
const PlainifySchema = `{
  "type": "object",
  "properties": {
    "plain": {"type": "string"}
  },
  "required": ["plain"]
}`

// plainifyPreamble opens every plainify system prompt.
const plainifyPreamble = "You are an expert editor. Rewrite the given text in plain English that a reader outside the project can follow on one pass. Apply all of these principles:"

// plainifyPrinciples are the standing contract: the rewrite says what the
// original said, in words a general reader already knows.
var plainifyPrinciples = []string{
	"- Keep every fact, name, number, and file path exactly as the original has it.",
	"- Write short sentences. One idea per sentence.",
	"- Use everyday words. Where the original reaches for jargon, say the plain thing it means.",
	"- Leave fenced code blocks and inline code spans byte for byte unchanged.",
	"- Add nothing the original does not say, and drop nothing it does.",
	"- Return the rewrite alone: no preamble, no labels, no commentary.",
}

// PlainifyConstraints are the caller's limits on a rewrite. MaxWords and
// Forbid reach the model as instructions and are also graded afterwards, so
// a rewrite that ignores them is caught rather than trusted.
type PlainifyConstraints struct {
	// MaxWords bounds the rewrite's whitespace-separated word count; zero
	// means no bound.
	MaxWords int
	// Forbid are patterns the rewrite must not match, such as the register
	// ids a reader outside the project cannot resolve.
	Forbid []*regexp.Regexp
	// NameByTitle asks the model to name referenced items by their titles
	// rather than their identifiers.
	NameByTitle bool
	// Glossary maps an identifier to the title that names it.
	Glossary map[string]string
}

// PlainViolation is one match of a --forbid pattern that survived the retry.
type PlainViolation struct {
	Pattern string `json:"pattern"`
	Match   string `json:"match"`
}

// PlainResult is a graded rewrite: the text plus what it did with the
// constraints. Truncated reports that the rewrite still runs past MaxWords,
// and Violations carries every forbidden match still in it.
type PlainResult struct {
	Plain      string
	Words      int
	Truncated  bool
	Violations []PlainViolation
}

// PlainEntry is one element of the array `plainify --json` reads.
type PlainEntry struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// BuildPlainifySystemPrompt renders the plain-English contract plus whatever
// constraints the caller set.
func BuildPlainifySystemPrompt(c PlainifyConstraints) string {
	base := plainifyPreamble + "\n" + strings.Join(plainifyPrinciples, "\n")
	directives := c.directives()
	if len(directives) == 0 {
		return base
	}
	return base + "\n\nThe rewrite also has to meet these constraints:\n" + strings.Join(directives, "\n")
}

func (c PlainifyConstraints) directives() []string {
	var out []string
	if c.MaxWords > 0 {
		out = append(out, fmt.Sprintf("- Use at most %d words.", c.MaxWords))
	}
	for _, re := range c.Forbid {
		out = append(out, fmt.Sprintf("- No part of the rewrite may match the regular expression %s.", re.String()))
	}
	if c.NameByTitle {
		out = append(out, "- Name every item you refer to by its title. Never print its identifier.")
	}
	if len(c.Glossary) > 0 {
		out = append(out, "- The titles for the identifiers in the original:")
		for _, id := range sortedKeys(c.Glossary) {
			out = append(out, fmt.Sprintf("  - %s is titled %q.", id, c.Glossary[id]))
		}
	}
	return out
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// grade measures a candidate rewrite against the constraints.
func (c PlainifyConstraints) grade(plain string) PlainResult {
	violations := []PlainViolation{}
	for _, re := range c.Forbid {
		for _, match := range re.FindAllString(plain, -1) {
			violations = append(violations, PlainViolation{Pattern: re.String(), Match: match})
		}
	}
	words := len(strings.Fields(plain))
	return PlainResult{
		Plain:      plain,
		Words:      words,
		Truncated:  c.MaxWords > 0 && words > c.MaxWords,
		Violations: violations,
	}
}

// retryDirectives names what a graded rewrite got wrong, one line per miss,
// for the second attempt's system prompt. An empty result means the rewrite
// met every constraint.
func (r PlainResult) retryDirectives(c PlainifyConstraints) []string {
	var out []string
	if r.Truncated {
		out = append(out, fmt.Sprintf("- Your last attempt ran to %d words against a limit of %d. Cut it to at most %d.", r.Words, c.MaxWords, c.MaxWords))
	}
	for _, v := range r.Violations {
		out = append(out, fmt.Sprintf("- Your last attempt matched the forbidden pattern %s at %q. Say that another way.", v.Pattern, v.Match))
	}
	return out
}

type plainifyEnvelope struct {
	Plain string `json:"plain"`
}

// Plainify sends text through claude with the plain-English contract. A
// rewrite that misses a constraint is retried once with its misses named,
// and the second attempt's grade is what comes back, met or not.
func Plainify(ctx context.Context, text string, c PlainifyConstraints, opts Options) (PlainResult, error) {
	cfg := opts.config(spawnllm.ProviderClaude, DefaultRewriteModel, DefaultRewriteTimeout)
	system := BuildPlainifySystemPrompt(c)

	plain, err := callPlainify(ctx, cfg, system, text)
	if err != nil {
		return PlainResult{}, err
	}
	graded := c.grade(plain)
	misses := graded.retryDirectives(c)
	if len(misses) == 0 {
		return graded, nil
	}

	tighter := system + "\n\nYour last attempt broke these constraints. Fix them, and change nothing else:\n" + strings.Join(misses, "\n")
	plain, err = callPlainify(ctx, cfg, tighter, text)
	if err != nil {
		return PlainResult{}, err
	}
	return c.grade(plain), nil
}

// PlainifyBatch plainifies each entry on its own call and returns the results
// in the order the entries arrived. One failed entry fails the batch.
func PlainifyBatch(ctx context.Context, entries []PlainEntry, c PlainifyConstraints, opts Options) ([]PlainResult, error) {
	results := make([]PlainResult, len(entries))
	errs := make([]error, len(entries))
	slots := make(chan struct{}, plainifyConcurrency)

	var wg sync.WaitGroup
	for i, entry := range entries {
		wg.Add(1)
		go func(i int, entry PlainEntry) {
			defer wg.Done()
			slots <- struct{}{}
			defer func() { <-slots }()
			results[i], errs[i] = Plainify(ctx, entry.Text, c, opts)
		}(i, entry)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			return nil, fmt.Errorf("entry %s: %w", entries[i].ID, err)
		}
	}
	return results, nil
}

func callPlainify(ctx context.Context, cfg Config, system, text string) (string, error) {
	var env plainifyEnvelope
	if err := RunSchema(ctx, cfg, system, text, json.RawMessage(PlainifySchema), &env); err != nil {
		return "", err
	}
	return strings.TrimSpace(env.Plain), nil
}
