// Package jslang masks JavaScript / TypeScript / JSX / TSX source so
// slop-cop's prose detectors only run on comment bodies, string-literal
// fragments, template-literal quasis, and JSX text children. Parsing is
// done with tree-sitter so the tricky disambiguations (regex vs division,
// JSX vs generics, nested template literals) are the grammar's problem and
// not ours.
//
// Byte offsets in the returned masked copy still index the original source,
// matching the contract of internal/markdown and internal/htmllang.
package jslang

import (
	"fmt"
	"strings"
	"unsafe"

	sitter "github.com/tree-sitter/go-tree-sitter"
	tsjs "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	tsts "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
	"github.com/yasyf/slop-cop/internal/lang"
	"github.com/yasyf/slop-cop/internal/types"
)

// Mode selects one of the four supported source dialects.
type Mode int

const (
	ModeJS Mode = iota
	ModeJSX
	ModeTS
	ModeTSX
)

func (m Mode) String() string {
	switch m {
	case ModeJS:
		return "js"
	case ModeJSX:
		return "jsx"
	case ModeTS:
		return "ts"
	case ModeTSX:
		return "tsx"
	}
	return fmt.Sprintf("unknown(%d)", int(m))
}

// analyzer is the lang.Analyzer implementation; one instance per mode. It
// carries no mutable state beyond the language pointer, which tree-sitter
// treats as read-only, so sharing across callers is safe. Each call
// allocates a fresh parser — tree-sitter's Parser is not goroutine-safe.
type analyzer struct {
	mode     Mode
	name     string
	language *sitter.Language
}

func newAnalyzer(mode Mode) *analyzer {
	return &analyzer{
		mode:     mode,
		name:     mode.String(),
		language: sitter.NewLanguage(languageFor(mode)),
	}
}

func languageFor(mode Mode) unsafe.Pointer {
	switch mode {
	case ModeJS, ModeJSX:
		return tsjs.Language()
	case ModeTS:
		return tsts.LanguageTypescript()
	case ModeTSX:
		return tsts.LanguageTSX()
	}
	panic(fmt.Sprintf("jslang: unknown mode %d", mode))
}

func (a *analyzer) Name() string { return a.name }

func (a *analyzer) Analyze(src string) (string, []lang.Range, error) {
	masked := buildMaskedTemplate(src)

	parser := sitter.NewParser()
	defer parser.Close()
	// language is a *sitter.Language; SetLanguage accepts it. The only
	// failure mode is an ABI mismatch between tree-sitter and the grammar
	// package, which we surface as an error rather than crashing.
	if err := parser.SetLanguage(a.language); err != nil {
		return "", nil, fmt.Errorf("jslang(%s): set language: %w", a.name, err)
	}
	tree := parser.Parse([]byte(src), nil)
	if tree == nil {
		return "", nil, fmt.Errorf("jslang(%s): parser returned nil tree", a.name)
	}
	defer tree.Close()

	var suppress []lang.Range
	walk(tree.RootNode(), func(n *sitter.Node) bool {
		start, end := int(n.StartByte()), int(n.EndByte())
		if start >= end || end > len(src) {
			return true
		}
		switch n.Kind() {
		case "comment":
			unmask(masked, src, start, end)
			kind := lang.KindComment
			if strings.HasPrefix(src[start:end], "/**") {
				kind = lang.KindJSDoc
			}
			suppress = append(suppress, lang.Range{Start: start, End: end, Kind: kind})
			// Comments are leaves — no children worth visiting.
			return false
		case "string_fragment":
			unmask(masked, src, start, end)
			suppress = append(suppress, lang.Range{Start: start, End: end, Kind: lang.KindStringLiteral})
			return false
		case "jsx_text":
			unmask(masked, src, start, end)
			suppress = append(suppress, lang.Range{Start: start, End: end, Kind: lang.KindJSXText})
			return false
		}
		return true
	})

	return string(masked), suppress, nil
}

func (a *analyzer) ApplySuppressions(vs []types.Violation, suppress []lang.Range, original string) []types.Violation {
	out := make([]types.Violation, 0, len(vs))
	for _, v := range vs {
		if v.RuleID == "dramatic-fragment" && lang.Overlaps(v.StartIndex, v.EndIndex, suppress, lang.KindJSDoc) {
			// JSDoc tag lines ("@param … name - a short phrase") look like
			// dramatic fragments to the detector; they're not.
			continue
		}
		lang.RestoreMatchedText(&v, original)
		out = append(out, v)
	}
	return out
}

// buildMaskedTemplate returns a []byte the same length as src where every
// byte is an ASCII space except for newlines, which are preserved at their
// original offsets. Callers unmask prose regions into this buffer by
// copying src bytes back over the spaces.
func buildMaskedTemplate(src string) []byte {
	buf := make([]byte, len(src))
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' {
			buf[i] = '\n'
		} else {
			buf[i] = ' '
		}
	}
	return buf
}

// unmask copies src[start:end] back into buf at the same offsets. Out-of-
// range indices are clamped so a malformed node doesn't panic.
func unmask(buf []byte, src string, start, end int) {
	if start < 0 {
		start = 0
	}
	if end > len(src) {
		end = len(src)
	}
	if end > len(buf) {
		end = len(buf)
	}
	for i := start; i < end; i++ {
		buf[i] = src[i]
	}
}

// walk visits n and every descendant in pre-order. The visitor returns true
// to descend into n's children, false to skip them. Uses direct Child()
// indexing rather than a TreeCursor — simpler, fast enough for our inputs,
// and avoids having to thread a cursor through recursion.
func walk(n *sitter.Node, visit func(*sitter.Node) bool) {
	if n == nil {
		return
	}
	if !visit(n) {
		return
	}
	count := n.ChildCount()
	for i := uint(0); i < count; i++ {
		walk(n.Child(i), visit)
	}
}

func init() {
	lang.Register(newAnalyzer(ModeJS), ".js", ".mjs", ".cjs")
	lang.Register(newAnalyzer(ModeJSX), ".jsx")
	lang.Register(newAnalyzer(ModeTS), ".ts", ".mts", ".cts")
	lang.Register(newAnalyzer(ModeTSX), ".tsx")
}
