// Package htmllang masks HTML source so slop-cop's prose detectors only run
// on element text content. Tags, attribute bodies, doctypes, comments, and
// the bodies of raw-text elements (<script>, <style>, <pre>, <code>,
// <template>) are overwritten with ASCII spaces while preserving newlines
// and byte offsets — the same contract honoured by internal/markdown.
package htmllang

import (
	"io"
	"strings"

	"github.com/yasyf/slop-cop/internal/lang"
	"github.com/yasyf/slop-cop/internal/types"
	"golang.org/x/net/html"
)

// Analyzer is the HTML implementation of lang.Analyzer.
type Analyzer struct{}

func (Analyzer) Name() string { return "html" }

// rawTextTags names elements whose text content should be masked even though
// the tokenizer emits it as a TextToken. Covers HTML5 raw-text elements plus
// <pre>/<code>/<template> where the contents are either code or presented
// verbatim — neither is "prose" for slop-cop's purposes.
var rawTextTags = map[string]bool{
	"script":   true,
	"style":    true,
	"pre":      true,
	"code":     true,
	"template": true,
	"textarea": true,
	"title":    true,
}

var headingTags = map[string]bool{
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
}

func (Analyzer) Analyze(src string) (string, []lang.Range, error) {
	buf := []byte(src)
	masked := append([]byte(nil), buf...)
	var suppress []lang.Range

	z := html.NewTokenizer(strings.NewReader(src))
	// Set a generous buffer cap so pathological single-element inputs don't
	// hit the default 1MB limit mid-parse on real-world HTML dumps.
	z.SetMaxBuf(0)

	offset := 0
	var rawDepth int          // >0 when the current text is inside script/style/pre/code/...
	var headingDepth int      // >0 when currently inside h1-h6
	var liDepth int           // >0 when currently inside <li>
	for {
		tt := z.Next()
		raw := z.Raw()
		start := offset
		end := offset + len(raw)
		offset = end

		if tt == html.ErrorToken {
			if z.Err() == io.EOF {
				break
			}
			// Malformed HTML: mask the remainder defensively and stop.
			maskSpan(masked, start, end)
			break
		}

		switch tt {
		case html.TextToken:
			// Keep prose text unless we're inside a raw-text container.
			if rawDepth > 0 {
				maskSpan(masked, start, end)
				if end > start {
					suppress = append(suppress, lang.Range{Start: start, End: end, Kind: lang.KindCodeBlock})
				}
				continue
			}
			if headingDepth > 0 && end > start {
				suppress = append(suppress, lang.Range{Start: start, End: end, Kind: lang.KindHeading})
			}
			if liDepth > 0 && end > start {
				suppress = append(suppress, lang.Range{Start: start, End: end, Kind: lang.KindListItem})
			}
			// Prose stays visible; no mutation of masked.
		case html.StartTagToken:
			maskSpan(masked, start, end)
			name, _ := z.TagName()
			n := strings.ToLower(string(name))
			if rawTextTags[n] {
				rawDepth++
			}
			if headingTags[n] {
				headingDepth++
			}
			if n == "li" {
				liDepth++
			}
		case html.EndTagToken:
			maskSpan(masked, start, end)
			name, _ := z.TagName()
			n := strings.ToLower(string(name))
			if rawTextTags[n] && rawDepth > 0 {
				rawDepth--
			}
			if headingTags[n] && headingDepth > 0 {
				headingDepth--
			}
			if n == "li" && liDepth > 0 {
				liDepth--
			}
		case html.SelfClosingTagToken, html.CommentToken, html.DoctypeToken:
			maskSpan(masked, start, end)
		}
	}

	return string(masked), suppress, nil
}

func (Analyzer) ApplySuppressions(vs []types.Violation, suppress []lang.Range, original string) []types.Violation {
	out := make([]types.Violation, 0, len(vs))
	for _, v := range vs {
		switch v.RuleID {
		case "dramatic-fragment":
			if lang.Overlaps(v.StartIndex, v.EndIndex, suppress, lang.KindHeading) {
				continue
			}
		case "staccato-burst":
			if lang.CountOverlapping(v.StartIndex, v.EndIndex, suppress, lang.KindListItem) >= 2 {
				continue
			}
		}
		lang.RestoreMatchedText(&v, original)
		out = append(out, v)
	}
	return out
}

// maskSpan overwrites buf[start:end] with ASCII space (0x20), preserving
// '\n' so line-based structure still matches the original.
func maskSpan(buf []byte, start, end int) {
	if start < 0 {
		start = 0
	}
	if end > len(buf) {
		end = len(buf)
	}
	for i := start; i < end; i++ {
		if buf[i] != '\n' {
			buf[i] = ' '
		}
	}
}

func init() {
	lang.Register(Analyzer{}, ".html", ".htm")
}
