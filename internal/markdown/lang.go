package markdown

import (
	"github.com/yasyf/slop-cop/internal/lang"
	"github.com/yasyf/slop-cop/internal/types"
)

// Analyzer is the markdown implementation of lang.Analyzer. Exported so
// callers can construct one directly; production code should go through
// the lang registry instead (lang.ByName / lang.ByExtension).
type Analyzer struct{}

func (Analyzer) Name() string { return "markdown" }

func (Analyzer) Analyze(src string) (string, []lang.Range, error) {
	masked, suppress, _ := Analyze(src)
	return masked, suppress, nil
}

func (Analyzer) ApplySuppressions(vs []types.Violation, suppress []lang.Range, original string) []types.Violation {
	return ApplySuppressions(vs, suppress, original)
}

func init() {
	lang.Register(Analyzer{}, ".md", ".markdown", ".mdx")
}
