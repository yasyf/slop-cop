// Package markdown provides a CommonMark-aware preprocessor for the slop-cop
// detectors. Analyze turns a markdown document into:
//
//   - a byte-identical-length "masked" string where non-prose regions (code,
//     URLs in links, raw HTML, YAML front matter) have been overwritten with
//     ASCII spaces while preserving newlines. Detectors run against this
//     string so regex patterns never match inside code or URLs.
//   - a list of structural ranges (ATX/setext headings, list items, code
//     blocks) used by the CLI's post-filter to suppress detector hits that
//     are false positives on markdown structure (e.g. `dramatic-fragment`
//     firing on `## Usage`).
//
// The implementation targets goldmark v1.8+ (where every AST node exposes
// Pos()), so link/image destination masking is done via a small CommonMark
// scanner rather than regex guesswork.
package markdown

import (
	"regexp"
	"sort"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// RangeKind identifies what a structural suppress Range corresponds to.
type RangeKind int

const (
	// KindHeading covers ATX (`# ...`) and setext (`Title\n===`) headings.
	KindHeading RangeKind = iota + 1
	// KindListItem covers a single item of an ordered or unordered list.
	KindListItem
	// KindCodeBlock covers fenced or indented code blocks. Emitted in
	// addition to masking so callers can distinguish "inside code" hits.
	KindCodeBlock
)

// Range is a half-open byte span [Start, End) into the source.
type Range struct {
	Start int       `json:"start"`
	End   int       `json:"end"`
	Kind  RangeKind `json:"kind,omitempty"`
}

// frontMatterRe matches a leading YAML front matter block (`---` fenced).
// Non-greedy body, optional trailing newline after the closing fence.
var frontMatterRe = regexp.MustCompile(`(?s)\A---\r?\n.*?\r?\n---[ \t]*(\r?\n|\z)`)

// Analyze parses src as CommonMark and returns the masked string, the list of
// structural suppress ranges (sorted by Start then End), and the byte range
// of any YAML front matter (Start == -1 when absent).
//
// Invariants on the return value:
//   - len(masked) == len(src)
//   - every '\n' in src is preserved at the same offset in masked
//   - bytes outside any mask range are byte-identical to src
//   - masked bytes are ASCII space (0x20), except preserved newlines
func Analyze(src string) (masked string, suppress []Range, frontMatter Range) {
	frontMatter = Range{Start: -1, End: -1}
	input := []byte(src)
	// buf is the byte buffer we'll return as `masked` after overwriting
	// non-prose ranges in place.
	buf := append([]byte(nil), input...)
	// parseBuf is the copy we hand to goldmark; we pre-blank the front matter
	// so goldmark's parser doesn't mistake `---` for a setext-heading underline
	// or thematic break.
	parseBuf := append([]byte(nil), input...)

	if loc := frontMatterRe.FindIndex(input); loc != nil && loc[0] == 0 {
		frontMatter = Range{Start: loc[0], End: loc[1]}
		maskSpan(buf, loc[0], loc[1])
		maskSpan(parseBuf, loc[0], loc[1])
	}

	reader := text.NewReader(parseBuf)
	doc := goldmark.DefaultParser().Parse(reader)

	var masks []Range
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node := n.(type) {
		case *ast.FencedCodeBlock:
			r := fencedBlockRange(node, parseBuf)
			if r.End > r.Start {
				masks = append(masks, r)
				suppress = append(suppress, Range{Start: r.Start, End: r.End, Kind: KindCodeBlock})
			}
		case *ast.CodeBlock:
			if r, ok := blockLinesRange(node); ok {
				masks = append(masks, r)
				suppress = append(suppress, Range{Start: r.Start, End: r.End, Kind: KindCodeBlock})
			}
		case *ast.HTMLBlock:
			if r, ok := htmlBlockRange(node); ok {
				masks = append(masks, r)
			}
		case *ast.CodeSpan:
			masks = append(masks, codeSpanRanges(node, parseBuf)...)
		case *ast.RawHTML:
			if node.Segments != nil {
				for i := 0; i < node.Segments.Len(); i++ {
					seg := node.Segments.At(i)
					masks = append(masks, Range{Start: seg.Start, End: seg.Stop})
				}
			}
		case *ast.AutoLink:
			if r, ok := autoLinkRange(node, parseBuf); ok {
				masks = append(masks, r)
			}
		case *ast.Link:
			if node.Reference != nil {
				break
			}
			if r, ok := linkDestRange(node, parseBuf); ok {
				masks = append(masks, r)
			}
		case *ast.Image:
			if node.Reference != nil {
				break
			}
			if r, ok := linkDestRange(node, parseBuf); ok {
				masks = append(masks, r)
			}
		case *ast.LinkReferenceDefinition:
			if r, ok := blockLinesRange(node); ok {
				masks = append(masks, r)
			}
		case *ast.Heading:
			if r, ok := headingRange(node, parseBuf); ok {
				suppress = append(suppress, Range{Start: r.Start, End: r.End, Kind: KindHeading})
			}
		case *ast.ListItem:
			if r, ok := listItemRange(node, parseBuf); ok {
				suppress = append(suppress, Range{Start: r.Start, End: r.End, Kind: KindListItem})
			}
		}
		return ast.WalkContinue, nil
	})

	for _, r := range masks {
		maskSpan(buf, r.Start, r.End)
	}

	sort.Slice(suppress, func(i, j int) bool {
		if suppress[i].Start != suppress[j].Start {
			return suppress[i].Start < suppress[j].Start
		}
		if suppress[i].End != suppress[j].End {
			return suppress[i].End < suppress[j].End
		}
		return suppress[i].Kind < suppress[j].Kind
	})

	return string(buf), suppress, frontMatter
}

// Overlaps reports whether the violation range [vStart, vEnd) overlaps any
// Range in spans whose Kind matches want. Used by the CLI to implement
// --markdown post-filtering without pulling detector logic into this package.
func Overlaps(vStart, vEnd int, spans []Range, want RangeKind) bool {
	for _, s := range spans {
		if s.Kind != want {
			continue
		}
		if vStart < s.End && vEnd > s.Start {
			return true
		}
	}
	return false
}

// CountOverlapping returns how many ranges of the given kind the violation
// span touches. Used by the staccato-burst suppression, which only drops a
// hit when it straddles two or more consecutive list items.
func CountOverlapping(vStart, vEnd int, spans []Range, want RangeKind) int {
	n := 0
	for _, s := range spans {
		if s.Kind != want {
			continue
		}
		if vStart < s.End && vEnd > s.Start {
			n++
		}
	}
	return n
}

// maskSpan overwrites buf[start:end] with ASCII space (0x20), preserving any
// '\n' in the range so paragraph boundaries remain intact for downstream
// detectors.
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

// linesBounds returns the [first.Start, last.Stop) span of a block node's
// `Lines()` or false if the node has no recorded lines.
func linesBounds(ls *text.Segments) (int, int, bool) {
	if ls == nil || ls.Len() == 0 {
		return 0, 0, false
	}
	first := ls.At(0)
	last := ls.At(ls.Len() - 1)
	return first.Start, last.Stop, true
}

// blockLinesRange works for any block node whose body corresponds to the
// union of its Lines segments (CodeBlock, LinkReferenceDefinition).
func blockLinesRange(n ast.Node) (Range, bool) {
	type linesNode interface{ Lines() *text.Segments }
	ln, ok := n.(linesNode)
	if !ok {
		return Range{}, false
	}
	s, e, has := linesBounds(ln.Lines())
	if !has {
		return Range{}, false
	}
	return Range{Start: s, End: e}, true
}

// fencedBlockRange returns the full span of a fenced code block, including
// the opening fence line and (if present) the closing fence line.
//
// goldmark's FencedCodeBlock.Lines() covers only the code body (between the
// opening and closing fences). We widen backward one line to pick up the
// opening fence, and forward up to two lines to pick up the closing fence.
func fencedBlockRange(n *ast.FencedCodeBlock, src []byte) Range {
	// Default to Pos() when the parser recorded one; otherwise derive from
	// the first content line and walk back one line to the opening fence.
	start := n.Pos()
	firstLineStart := -1
	if s, _, ok := linesBounds(n.Lines()); ok {
		firstLineStart = lineStart(src, s)
	}
	if start < 0 || start > firstLineStart && firstLineStart >= 0 {
		start = firstLineStart
	}
	start = lineStart(src, start)
	// If start sits on the first content line (not on a fence line), back
	// up one line so the opening fence gets masked too.
	if firstLineStart >= 0 && start == firstLineStart && start > 0 {
		prevEnd := start - 1 // the '\n' separating fence line from content
		prevStart := lineStart(src, prevEnd)
		if isOpeningFence(src[prevStart:prevEnd]) {
			start = prevStart
		}
	}
	// End defaults to end of the last content line; scan past any closing
	// fence that appears on a following line.
	end := start
	if _, e, ok := linesBounds(n.Lines()); ok {
		end = e
	}
	end = scanPastClosingFence(src, end)
	return Range{Start: start, End: end}
}

// isOpeningFence reports whether a line is a code-fence opener: optional
// up-to-3-space indent, >=3 backticks or tildes, then anything until EOL.
func isOpeningFence(line []byte) bool {
	i := 0
	for i < len(line) && line[i] == ' ' && i < 3 {
		i++
	}
	if i >= len(line) {
		return false
	}
	fc := line[i]
	if fc != '`' && fc != '~' {
		return false
	}
	run := 0
	for i < len(line) && line[i] == fc {
		i++
		run++
	}
	return run >= 3
}

// scanPastClosingFence advances `pos` past a closing code fence. It treats
// the line containing `pos` itself as a candidate first (goldmark's
// FencedCodeBlock.Lines() last Stop may sit at the start of the closing
// fence line, or one past the trailing newline of the final content line).
// Returns pos unchanged if no closing fence is found within the next 3 lines.
func scanPastClosingFence(src []byte, pos int) int {
	if pos >= len(src) {
		return len(src)
	}
	// Find the start of the line containing pos.
	i := lineStart(src, pos)
	for try := 0; try < 3 && i < len(src); try++ {
		lineEnd := i
		for lineEnd < len(src) && src[lineEnd] != '\n' {
			lineEnd++
		}
		line := src[i:lineEnd]
		if isClosingFence(line) {
			if lineEnd < len(src) {
				lineEnd++ // include trailing newline
			}
			return lineEnd
		}
		i = lineEnd
		if i < len(src) {
			i++
		}
	}
	return pos
}

// isClosingFence reports whether a single line is a fenced-code closing
// marker (optional leading indent + >=3 backticks or tildes + optional
// trailing spaces).
func isClosingFence(line []byte) bool {
	i := 0
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') && i < 3 {
		i++
	}
	if i >= len(line) {
		return false
	}
	fc := line[i]
	if fc != '`' && fc != '~' {
		return false
	}
	run := 0
	for i < len(line) && line[i] == fc {
		i++
		run++
	}
	if run < 3 {
		return false
	}
	for i < len(line) {
		if line[i] != ' ' && line[i] != '\t' {
			return false
		}
		i++
	}
	return true
}

// htmlBlockRange covers the full HTML block including its closure line.
func htmlBlockRange(n *ast.HTMLBlock) (Range, bool) {
	s, e, ok := linesBounds(n.Lines())
	if !ok && !n.HasClosure() {
		return Range{}, false
	}
	if n.HasClosure() {
		cl := n.ClosureLine
		if !ok {
			s = cl.Start
			e = cl.Stop
		} else {
			if cl.Start < s {
				s = cl.Start
			}
			if cl.Stop > e {
				e = cl.Stop
			}
		}
	}
	return Range{Start: s, End: e}, true
}

// codeSpanRanges masks the inline code span and its surrounding backticks.
// goldmark gives us the inner Text segments; we widen to cover the ticks.
func codeSpanRanges(n *ast.CodeSpan, src []byte) []Range {
	var out []Range
	start := n.Pos()
	end := start
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			if t.Segment.Stop > end {
				end = t.Segment.Stop
			}
		}
	}
	// Widen to include opening backticks (backtrack until non-backtick) and
	// closing backticks (advance until non-backtick).
	for start > 0 && src[start-1] == '`' {
		start--
	}
	for end < len(src) && src[end] == '`' {
		end++
	}
	if end > start {
		out = append(out, Range{Start: start, End: end})
	}
	return out
}

// autoLinkRange covers the entire `<url>` or `<email>` inline span.
func autoLinkRange(n *ast.AutoLink, src []byte) (Range, bool) {
	start := n.Pos()
	if start < 0 || start >= len(src) {
		return Range{}, false
	}
	if src[start] != '<' {
		// Pos landed on the URL body for some parses; back up to the `<`.
		for start > 0 && src[start-1] != '<' {
			start--
		}
		if start > 0 {
			start--
		}
	}
	if start >= len(src) || src[start] != '<' {
		return Range{}, false
	}
	end := start + 1
	for end < len(src) && src[end] != '>' && src[end] != '\n' {
		end++
	}
	if end < len(src) && src[end] == '>' {
		end++
	}
	return Range{Start: start, End: end}, true
}

// linkDestRange locates the destination `(...)` range of an inline link or
// image. It accepts any ast.Node (Link or Image) and returns false for
// reference-style links, shortcut links, or malformed parses.
func linkDestRange(n ast.Node, src []byte) (Range, bool) {
	// Find the byte offset of the `]` that closes the link text by walking
	// the link's last descendant that exposes a segment.
	closeBracket := findLinkTextEnd(n, src)
	if closeBracket < 0 || closeBracket >= len(src) || src[closeBracket] != ']' {
		return Range{}, false
	}
	openParen := closeBracket + 1
	if openParen >= len(src) || src[openParen] != '(' {
		return Range{}, false
	}
	endParen, ok := scanInlineLinkDestination(src, openParen)
	if !ok {
		return Range{}, false
	}
	return Range{Start: openParen, End: endParen + 1}, true
}

// findLinkTextEnd returns the byte offset of the `]` that closes the link's
// text, by scanning forward from the link's Pos() with bracket-depth
// tracking. Code spans, autolinks, and image-form `![alt](url)` subspans
// are skipped so their internal brackets don't confuse the depth counter.
// Returns -1 if the closing bracket can't be located.
func findLinkTextEnd(n ast.Node, src []byte) int {
	start := n.Pos()
	if start < 0 {
		return -1
	}
	// Image nodes may report Pos at the leading `!` of `![...`; step into
	// the opening bracket when we see that.
	if start+1 < len(src) && src[start] == '!' && src[start+1] == '[' {
		start++
	}
	if start >= len(src) || src[start] != '[' {
		return -1
	}
	i := start + 1
	depth := 1
	for i < len(src) {
		c := src[i]
		switch c {
		case '\\':
			if i+1 < len(src) {
				i += 2
				continue
			}
		case '`':
			// Skip balanced code span: count opening backticks, advance past
			// the matching closing run. If unbalanced, fall through.
			j := i
			for j < len(src) && src[j] == '`' {
				j++
			}
			run := j - i
			k := j
			for k < len(src) {
				if src[k] == '`' {
					cj := k
					for cj < len(src) && src[cj] == '`' {
						cj++
					}
					if cj-k == run {
						k = cj
						break
					}
					k = cj
					continue
				}
				k++
			}
			i = k
			continue
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return i
			}
		}
		i++
	}
	return -1
}

// scanInlineLinkDestination takes the offset of `(` in `](` and returns the
// offset of the matching `)`, following CommonMark section 6.3 rules:
//   - optional leading whitespace
//   - destination: either <...> (angle form) or a non-whitespace run with
//     balanced parens up to depth 1, and escaped parens via backslash
//   - optional whitespace + title ("...", '...', or (...))
//   - optional whitespace, then `)`
//
// Returns false if the balanced close isn't found before EOL or EOF.
func scanInlineLinkDestination(src []byte, openParen int) (int, bool) {
	i := openParen + 1
	n := len(src)
	skipSpaces := func() {
		for i < n && (src[i] == ' ' || src[i] == '\t') {
			i++
		}
	}
	skipSpaces()
	if i >= n {
		return 0, false
	}
	// Destination
	if src[i] == '<' {
		// Angle-bracket form: no newlines, no `<` or `>` inside.
		i++
		for i < n && src[i] != '>' && src[i] != '\n' && src[i] != '<' {
			if src[i] == '\\' && i+1 < n {
				i += 2
				continue
			}
			i++
		}
		if i >= n || src[i] != '>' {
			return 0, false
		}
		i++
	} else {
		// Bare destination: no ASCII control or space, balanced parens.
		depth := 0
		for i < n {
			c := src[i]
			if c == '\\' && i+1 < n {
				i += 2
				continue
			}
			if c == '(' {
				depth++
				i++
				continue
			}
			if c == ')' {
				if depth == 0 {
					break
				}
				depth--
				i++
				continue
			}
			if c == ' ' || c == '\t' || c == '\n' {
				break
			}
			i++
		}
	}
	// Optional title
	skipSpaces()
	if i < n {
		switch src[i] {
		case '"', '\'':
			quote := src[i]
			i++
			for i < n && src[i] != quote {
				if src[i] == '\\' && i+1 < n {
					i += 2
					continue
				}
				if src[i] == '\n' {
					// Titles may span lines per spec but we stay conservative.
					i++
					continue
				}
				i++
			}
			if i < n && src[i] == quote {
				i++
			}
		case '(':
			depth := 1
			i++
			for i < n && depth > 0 {
				if src[i] == '\\' && i+1 < n {
					i += 2
					continue
				}
				if src[i] == '(' {
					depth++
				} else if src[i] == ')' {
					depth--
					if depth == 0 {
						i++
						break
					}
				}
				i++
			}
		}
	}
	skipSpaces()
	if i >= n || src[i] != ')' {
		return 0, false
	}
	return i, true
}

// headingRange covers an ATX or setext heading block. For ATX headings the
// range is widened to the start of the line so it includes the `#` markers
// (goldmark's Lines() segments start at the heading text, not the markers).
// For setext headings the range includes the underline line when present.
func headingRange(n *ast.Heading, src []byte) (Range, bool) {
	s, e, ok := linesBounds(n.Lines())
	if !ok {
		return Range{}, false
	}
	// Widen start to include ATX `#` markers / leading whitespace / blockquote
	// prefix — anything up to the start of the line containing the text.
	s = lineStart(src, s)
	// Setext headings have the heading text in Lines() but the underline is
	// on the line immediately below. Extend to the end of that underline
	// line if it looks like `===` / `---`.
	extended := e
	for extended < len(src) && src[extended] == '\n' {
		extended++
	}
	lineEnd := extended
	for lineEnd < len(src) && src[lineEnd] != '\n' {
		lineEnd++
	}
	if isSetextUnderline(src[extended:lineEnd]) {
		if lineEnd < len(src) {
			lineEnd++
		}
		e = lineEnd
	}
	return Range{Start: s, End: e}, true
}

// isSetextUnderline reports whether a line is a valid setext heading
// underline: 1+ `=` or `-` chars with optional leading/trailing spaces.
func isSetextUnderline(line []byte) bool {
	i := 0
	for i < len(line) && line[i] == ' ' && i < 3 {
		i++
	}
	if i >= len(line) {
		return false
	}
	ch := line[i]
	if ch != '=' && ch != '-' {
		return false
	}
	run := 0
	for i < len(line) && line[i] == ch {
		i++
		run++
	}
	if run == 0 {
		return false
	}
	for i < len(line) {
		if line[i] != ' ' && line[i] != '\t' {
			return false
		}
		i++
	}
	return true
}

// listItemRange returns the byte span covering a list item, from its marker
// to the end of its last child block.
func listItemRange(n *ast.ListItem, src []byte) (Range, bool) {
	start := n.Pos()
	if start < 0 {
		start = n.Offset
	}
	if start < 0 {
		return Range{}, false
	}
	// Marker may appear before Pos() in goldmark (Pos points at the content
	// offset). Back up to the list marker (-, *, +, or digit).
	for start > 0 && src[start-1] != '\n' {
		start--
	}
	end := start
	// Collect the max end from descendants that expose a safe source span.
	// Block-level nodes carry line segments via Lines(); inline-level nodes
	// expose only per-node Segment fields (Text, RawHTML). Attempting to
	// call Lines() on an inline node panics ("can not call with inline
	// nodes"), so gate on Type() first.
	var walk func(ast.Node)
	walk = func(nn ast.Node) {
		if nn.Type() == ast.TypeBlock {
			if ln, ok := nn.(interface{ Lines() *text.Segments }); ok {
				if ls := ln.Lines(); ls != nil && ls.Len() > 0 {
					last := ls.At(ls.Len() - 1)
					if last.Stop > end {
						end = last.Stop
					}
				}
			}
		}
		if t, ok := nn.(*ast.Text); ok && t.Segment.Stop > end {
			end = t.Segment.Stop
		}
		for c := nn.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(n)
	if end <= start {
		return Range{}, false
	}
	return Range{Start: start, End: end}, true
}

// lineStart returns the offset of the start of the line containing pos.
func lineStart(src []byte, pos int) int {
	if pos < 0 {
		return 0
	}
	if pos > len(src) {
		pos = len(src)
	}
	for pos > 0 && src[pos-1] != '\n' {
		pos--
	}
	return pos
}
