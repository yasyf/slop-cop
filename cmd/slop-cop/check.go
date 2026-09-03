package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yasyf/slop-cop/internal/detectors"
	_ "github.com/yasyf/slop-cop/internal/htmllang" // lang registry
	_ "github.com/yasyf/slop-cop/internal/jslang"   // lang registry
	"github.com/yasyf/slop-cop/internal/lang"
	"github.com/yasyf/slop-cop/internal/llm"
	_ "github.com/yasyf/slop-cop/internal/markdown" // lang registry
	"github.com/yasyf/slop-cop/internal/readability"
	"github.com/yasyf/slop-cop/internal/rules"
	"github.com/yasyf/slop-cop/internal/types"
)

// checkReport is the JSON document emitted by `slop-cop check`. Counts are
// denormalised so callers don't have to walk the violations slice.
type checkReport struct {
	// Ran is always true. A consumer tests for the field's presence: a
	// killed or truncated run leaves output that cannot decode into a
	// report claiming it ran, so empty stdout never reads as a clean pass.
	Ran bool `json:"ran"`
	// Version and BinaryPath identify the build that produced the report,
	// so a rule-ID mismatch resolves against the binary on disk. Both keys
	// are always emitted; either value is empty when undeterminable, which
	// is not a signal about the run. Ran is.
	Version          string                          `json:"version"`
	BinaryPath       string                          `json:"binary_path"`
	TextLength       int                             `json:"text_length"`
	Violations       []types.Violation               `json:"violations"`
	CountsByRule     map[string]int                  `json:"counts_by_rule"`
	CountsByCategory map[types.ViolationCategory]int `json:"counts_by_category"`
	// Lang reports which input-language masker ran. "text" means no masking;
	// other values ("markdown", "html", "jsx", "tsx", "ts", "js") match the
	// registered Analyzer names in internal/lang.
	Lang string `json:"lang"`
	// LLMEffort is the effort level slop-cop resolved to for this run:
	// "off", "low", or "high". Emitted alongside LLM so agents can read
	// the setting even when no pass ran.
	LLMEffort string `json:"llm_effort"`
	// LLM captures per-tier outcomes. Omitted entirely when effort=off.
	LLM *llmReport `json:"llm,omitempty"`
	// Readability is advisory, never a violation: it carries no span, so it
	// is absent from Violations and both count maps, and --lines does not
	// filter it. Omitted when the base layer is off or the prose is too
	// short to score.
	Readability *readability.Report `json:"readability,omitempty"`
	// Rules carries the fix guidance for each rule that fired, once per
	// distinct rule ID rather than once per violation.
	Rules map[string]reportRule `json:"rules,omitempty"`
	// Config names the .slopcop.toml that filtered this run. Omitted when
	// no config applied.
	Config string `json:"config,omitempty"`
}

// reportRule is the fix guidance sidecar for one rule that fired.
type reportRule struct {
	Name        string                  `json:"name"`
	Category    types.ViolationCategory `json:"category"`
	Tip         string                  `json:"tip,omitempty"`
	RewriteHint string                  `json:"rewriteHint,omitempty"`
}

// llmReport captures the outcome of the two LLM passes. Either sub-field
// may be nil if that specific pass wasn't attempted.
type llmReport struct {
	Sentence *llmPassStatus `json:"sentence,omitempty"`
	Document *llmPassStatus `json:"document,omitempty"`
}

// llmPassStatus describes a single LLM pass's outcome.
type llmPassStatus struct {
	// Auto is true when the pass was enabled by the auto-default rather
	// than an explicit flag. Drives fail-open vs fail-closed semantics.
	Auto bool `json:"auto"`
	// Ran is true when the pass executed successfully and contributed
	// violations to the report.
	Ran bool `json:"ran"`
	// Error holds the pass's error message when Ran is false. Empty when
	// the pass succeeded or wasn't attempted.
	Error string `json:"error,omitempty"`
}

// resolveLang picks the lang.Analyzer for this invocation based on --lang.
// Returns the selected Analyzer (nil for "text") and the lang name to record
// in checkReport.Lang. "auto" detects from the file extension, falling back
// to "text" for stdin or an unrecognised extension. An unknown explicit
// value is a usage error.
func resolveLang(langFlag, path string) (lang.Analyzer, string, error) {
	pick := strings.ToLower(langFlag)
	if pick == "" {
		pick = "auto"
	}
	if pick == "auto" {
		if path == "" || path == "-" {
			return nil, "text", nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if a, ok := lang.ByExtension(ext); ok {
			return a, a.Name(), nil
		}
		return nil, "text", nil
	}
	a, ok := lang.ByName(pick)
	if !ok {
		return nil, "", fmt.Errorf("invalid --lang value %q (want auto|text|markdown|html|jsx|tsx|ts|js)", pick)
	}
	if a == nil {
		// "text" resolves to no analyzer; still a valid selection.
		return nil, "text", nil
	}
	return a, a.Name(), nil
}

// standardLayer selects which rule layers a run executes.
type standardLayer string

const (
	standardAll    standardLayer = "all"
	standardSlop   standardLayer = "slop"
	standardBase   standardLayer = "base"
	standardGoogle standardLayer = "google"
)

// resolveStandard maps --standard onto a layer selection. An unknown value is
// a usage error.
func resolveStandard(standardFlag string) (standardLayer, error) {
	pick := strings.ToLower(standardFlag)
	if pick == "" {
		pick = "all"
	}
	switch standardLayer(pick) {
	case standardAll, standardSlop, standardBase, standardGoogle:
		return standardLayer(pick), nil
	}
	return "", fmt.Errorf("invalid --standard value %q (want all|slop|base|google)", pick)
}

// dropSuperseded silences a rule on the spans a google rule already ruled on,
// so a run carrying both layers reports the style guide's verdict once instead
// of the looser rule's alongside it. Scoping to the overlap keeps the silenced
// rule live everywhere else in the document.
func dropSuperseded(vs []types.Violation) []types.Violation {
	silenced := map[string][]types.Violation{}
	for _, v := range vs {
		for _, id := range rules.Supersedes[v.RuleID] {
			silenced[id] = append(silenced[id], v)
		}
	}
	if len(silenced) == 0 {
		return vs
	}
	out := make([]types.Violation, 0, len(vs))
	for _, v := range vs {
		if overlapsAny(v, silenced[v.RuleID]) {
			continue
		}
		out = append(out, v)
	}
	return out
}

func overlapsAny(v types.Violation, spans []types.Violation) bool {
	for _, s := range spans {
		if v.StartIndex < s.EndIndex && s.StartIndex < v.EndIndex {
			return true
		}
	}
	return false
}

func (s standardLayer) runsSlop() bool   { return s == standardAll || s == standardSlop }
func (s standardLayer) runsBase() bool   { return s == standardAll || s == standardBase }
func (s standardLayer) runsGoogle() bool { return s == standardAll || s == standardGoogle }

// catalogue returns the rules this layer selection puts in front of the LLM
// passes. Slop first in All is load-bearing: buildRulePrompt numbers rules by
// position, so the slop-only prompt is the head of the combined one.
func (s standardLayer) catalogue() []types.ViolationRule {
	switch s {
	case standardSlop:
		return rules.Slop
	case standardBase:
		return rules.Base
	case standardGoogle:
		return rules.Google
	}
	return rules.All
}

// llmEffort is the canonical effort level for the LLM passes.
type llmEffort string

const (
	effortOff  llmEffort = "off"
	effortLow  llmEffort = "low"
	effortHigh llmEffort = "high"
)

// envLLMEffort names the environment variable that sets the effort level when
// no flag does.
const envLLMEffort = "SLOP_COP_LLM"

// autoEnableLLM returns true when the LLM passes should be auto-enabled for
// this invocation: the `codex` CLI is on $PATH.
func autoEnableLLM() bool {
	_, err := exec.LookPath("codex")
	return err == nil
}

// resolveEffort picks the LLM effort level for this run, in the precedence
// --llm-effort > --no-llm > --llm-deep > --llm > $SLOP_COP_LLM > auto. The
// second result is true when the level came from auto, which is what makes an
// auto-enabled pass fail open where an explicit one fails closed.
func resolveEffort(cmd *cobra.Command, effortFlag string, llmFlag, deepFlag, noLLMFlag bool) (llmEffort, bool, error) {
	if cmd.Flags().Changed("llm-effort") {
		eff, auto, err := parseEffort(effortFlag)
		if err != nil {
			return effortOff, false, fmt.Errorf("invalid --llm-effort %q (want off|low|high|auto)", effortFlag)
		}
		return eff, auto, nil
	}
	// --no-llm names an outcome rather than a tier, so it beats both tier
	// aliases; --llm-deep is more specific than --llm, so it wins there.
	if cmd.Flags().Changed("no-llm") && noLLMFlag {
		return effortOff, false, nil
	}
	if cmd.Flags().Changed("llm-deep") {
		if deepFlag {
			return effortHigh, false, nil
		}
		return effortOff, false, nil
	}
	if cmd.Flags().Changed("llm") {
		if llmFlag {
			return effortLow, false, nil
		}
		return effortOff, false, nil
	}
	if raw := strings.TrimSpace(os.Getenv(envLLMEffort)); raw != "" {
		eff, auto, err := parseEffort(raw)
		if err != nil {
			return effortOff, false, fmt.Errorf("invalid %s %q (want off|low|high|auto)", envLLMEffort, raw)
		}
		return eff, auto, nil
	}
	return autoEffort(), true, nil
}

// parseEffort maps an effort word onto a level. The second result is true for
// "auto", the one value resolved through the claude-availability default.
func parseEffort(raw string) (llmEffort, bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "off":
		return effortOff, false, nil
	case "low":
		return effortLow, false, nil
	case "high":
		return effortHigh, false, nil
	case "", "auto":
		return autoEffort(), true, nil
	}
	return effortOff, false, fmt.Errorf("want off|low|high|auto")
}

func autoEffort() llmEffort {
	if autoEnableLLM() {
		return effortLow
	}
	return effortOff
}

// ruleCounts totals the catalogue by tier so the help text cannot drift from
// the rules themselves. LLM rules are counted by RequiresLLM: false-range
// carries an LLMTier it does not use.
func ruleCounts() (client, sentence, document int) {
	for _, r := range rules.All {
		switch {
		case !r.RequiresLLM:
			client++
		case r.LLMTier == types.LLMTierDocument:
			document++
		default:
			sentence++
		}
	}
	return client, sentence, document
}

func newCheckCmd() *cobra.Command {
	var (
		llmFlag    bool
		deepFlag   bool
		noLLMFlag  bool
		effort     string
		sentModel  string
		docModel   string
		sentTO     time.Duration
		docTO      time.Duration
		langMode   string
		linesFlag  string
		standard   string
		configFlag string
		strictFlag bool
	)
	client, sentence, document := ruleCounts()
	cmd := &cobra.Command{
		Use:   "check [path|-]",
		Short: "Run detectors over a file (or stdin) and emit a JSON report.",
		Long: fmt.Sprintf(`Runs the %d client-side rules by default. Two optional LLM passes
run via the codex CLI:

  low   sentence-tier semantic analysis, %d rules
  high  low + document-tier structural analysis, %d rules

Choose one with --llm-effort (off|low|high|auto), or use the sugar aliases:
  --llm       → --llm-effort=low
  --llm-deep  → --llm-effort=high
  --no-llm    → --llm-effort=off

$SLOP_COP_LLM takes the same four values and applies when no flag does.
When the codex CLI is on $PATH, --llm-effort=auto resolves to 'low';
otherwise 'off'. Auto-enabled passes fail open (the
failure is reported under 'llm.<tier>.error' and the client-side results
are still returned); explicit passes propagate the error as exit code 3.

Input is taken from the path argument, or from stdin when the path is "-"
or omitted.

Language-aware mode masks non-prose regions of the input before detectors
run, so slop-cop flags prose only — not code, tags, URLs, or other syntax.
Pick a mode with --lang:

  auto      (default) pick from the file extension; "text" for stdin.
  text      no masking; treat input as plain prose.
  markdown  CommonMark; mask code fences, links, HTML, YAML front matter.
  html      HTML; mask tags, attributes, <script>/<style>/<pre>/<code>.
  jsx,tsx   JS/TS with JSX; keep comments, strings, template quasis, JSX text.
  ts,js     JS/TS without JSX.

Suppressions inside masked modes drop structural false positives
(e.g. 'dramatic-fragment' on headings, 'staccato-burst' across list items).

The %d rules come in three layers, all on by default. Pick with --standard:

  all     (default) every layer.
  slop    the original LLM-tell catalogue: does this sound like an LLM?
  base    the plain-language base layer: can this be read once and understood?
  google  the Google developer documentation style guide.

The selection applies to the LLM passes too — a layer left out never reaches
the prompt.

Individual rules are suppressed by a %s resolved by walking up from
the input path (--config overrides it):

  disable = ["colon-elaboration"]
  enable_only = ["overused-intensifiers"]

  [[overrides]]
  paths = ["docs/**/*.md"]
  disable = ["unformatted-code-identifier"]

Use --lines to report only the violations that begin within a 1-based inclusive
line range, while still scanning the whole document for context — useful for
linting just the lines an edit touched:

  50:80  a closed range          50:  from line 50 to EOF
  :80    from line 1 to line 80  50   a single line`,
			client, sentence, document, len(rules.All), configName),
		Example: `  slop-cop check article.md --pretty
  cat article.md | slop-cop check --no-llm
  slop-cop check component.tsx --lang=tsx --no-llm
  slop-cop check README.md --lines 50:80 --format=compact
  slop-cop check - --lang=markdown --llm-effort=high < article.md`,
		Args: cobra.MaximumNArgs(1),
	}
	pretty := addPrettyFlag(cmd)
	format := addFormatFlag(cmd)
	cmd.Flags().StringVar(&effort, "llm-effort", "auto", "LLM analysis effort: off|low|high|auto. Auto = low when the codex CLI is on $PATH, off otherwise. $SLOP_COP_LLM sets it when no flag does.")
	cmd.Flags().BoolVar(&llmFlag, "llm", false, "Alias for --llm-effort=low (sentence tier).")
	cmd.Flags().BoolVar(&deepFlag, "llm-deep", false, "Alias for --llm-effort=high (sentence + document tiers).")
	cmd.Flags().BoolVar(&noLLMFlag, "no-llm", false, "Alias for --llm-effort=off (client-side rules only). Wins over --llm and --llm-deep.")
	cmd.Flags().StringVar(&configFlag, "config", "", "Path to a "+configName+". Default: the nearest one above the input path; its absence is not an error.")
	cmd.Flags().BoolVar(&strictFlag, "strict", false, "Exit 1 when the report carries violations. Off by default: a completed run exits 0 whatever it found.")
	cmd.Flags().StringVar(&sentModel, "sentence-model", llm.DefaultSentenceModel, "Codex model slug for the sentence pass.")
	cmd.Flags().StringVar(&docModel, "document-model", llm.DefaultDocumentModel, "Codex model slug for the document pass.")
	cmd.Flags().DurationVar(&sentTO, "sentence-timeout", llm.DefaultSentenceTimeout, "Wall-clock bound on each sentence-pass chunk, retries included.")
	cmd.Flags().DurationVar(&docTO, "document-timeout", llm.DefaultDocumentTimeout, "Wall-clock bound on the document pass, retries included.")
	cmd.Flags().StringVar(&langMode, "lang", "auto", "Input language: auto|text|markdown|html|jsx|tsx|ts|js.")
	cmd.Flags().StringVar(&linesFlag, "lines", "", "Restrict the report to a 1-based inclusive line range, e.g. 50:80 (open-ended 50: or :80; a bare 50 is one line). Detectors still run over the full input.")
	cmd.Flags().StringVar(&standard, "standard", "all", "Rule layers to run: all (every layer), slop (the original LLM-tell catalogue), base (the plain-language base layer), google (the Google developer documentation style guide).")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		std, err := resolveStandard(standard)
		if err != nil {
			return usageError{err: err}
		}
		outFormat, err := resolveFormat(*format)
		if err != nil {
			return usageError{err: err}
		}

		path := pathArg(args)
		filter, configFile, err := resolveRuleFilter(configFlag, path)
		if err != nil {
			return err
		}

		text, err := readInput(path)
		if err != nil {
			return err
		}
		ctx := runContext(cmd)

		analyzer, langName, err := resolveLang(langMode, path)
		if err != nil {
			return usageError{err: err}
		}

		lr, hasLines, err := parseLineRange(linesFlag)
		if err != nil {
			return usageError{err: err}
		}

		eff, auto, err := resolveEffort(cmd, effort, llmFlag, deepFlag, noLLMFlag)
		if err != nil {
			return usageError{err: err}
		}
		runSentence := eff == effortLow || eff == effortHigh
		runDocument := eff == effortHigh

		// When a lang analyzer is selected, detectors (and the LLM passes)
		// run on a masked copy of the input. Offsets still index into the
		// original bytes (Analyze preserves length), so we re-slice
		// MatchedText from the original in ApplySuppressions.
		scanText := text
		var suppress []lang.Range
		if analyzer != nil {
			m, s, aerr := analyzer.Analyze(text)
			if aerr != nil {
				return fmt.Errorf("%s: analyze: %w", analyzer.Name(), aerr)
			}
			scanText, suppress = m, s
		}

		var violations []types.Violation
		if std.runsSlop() {
			violations = detectors.RunClient(scanText)
		}
		if std.runsBase() {
			violations = append(violations, detectors.RunBase(scanText)...)
		}
		if std.runsGoogle() {
			violations = append(violations, detectors.RunGoogle(scanText)...)
		}
		// Filtering precedes supersession so a disabled google rule cannot
		// silence a slop rule on its span from beyond the grave.
		violations = filter.keep(violations)
		if std.runsGoogle() {
			violations = dropSuperseded(violations)
		}

		catalogue := filter.catalogue(std.catalogue())

		var llmRep *llmReport
		ensureReport := func() *llmReport {
			if llmRep == nil {
				llmRep = &llmReport{}
			}
			return llmRep
		}

		if runSentence {
			opts := llm.Options{Model: sentModel, Timeout: sentTO, Rules: catalogue}
			vs, err := llm.RunSentence(ctx, scanText, opts)
			if err != nil {
				if auto {
					fmt.Fprintln(os.Stderr, "slop-cop: sentence LLM pass skipped (auto-enabled, codex failed):", err)
					ensureReport().Sentence = &llmPassStatus{Auto: true, Ran: false, Error: err.Error()}
				} else {
					return llmError{err: fmt.Errorf("sentence pass: %w", err)}
				}
			} else {
				violations = append(violations, vs...)
				ensureReport().Sentence = &llmPassStatus{Auto: auto, Ran: true}
			}
		}
		if runDocument {
			opts := llm.Options{Model: docModel, Timeout: docTO, Rules: catalogue}
			vs, err := llm.RunDocument(ctx, scanText, opts)
			if err != nil {
				if auto {
					fmt.Fprintln(os.Stderr, "slop-cop: document LLM pass skipped (auto-enabled, codex failed):", err)
					ensureReport().Document = &llmPassStatus{Auto: true, Ran: false, Error: err.Error()}
				} else {
					return llmError{err: fmt.Errorf("document pass: %w", err)}
				}
			} else {
				violations = append(violations, vs...)
				ensureReport().Document = &llmPassStatus{Auto: auto, Ran: true}
			}
		}
		// Unconditional: the two client layers share the elevated-register rule
		// ID, so a merged run can collide without any LLM pass having run.
		violations = detectors.Deduplicate(violations)
		violations = filter.keep(violations)

		if analyzer != nil {
			violations = analyzer.ApplySuppressions(violations, suppress, text)
		}

		sortViolations(violations)
		if hasLines {
			violations = filterByLines(violations, text, lr)
		}
		if violations == nil {
			violations = []types.Violation{}
		}

		// Scored over the masked text, so code spans and fences don't skew the
		// grade, and so the score describes the same prose the detectors saw.
		var read *readability.Report
		if std != standardSlop {
			read = readability.Analyze(scanText)
		}

		ver, _ := buildMetadata()
		// Best-effort, like the version beside it: binary_path is diagnostic,
		// so a process that cannot self-locate loses the field, not the run.
		binary, _ := os.Executable()

		report := checkReport{
			Ran:              true,
			Version:          ver,
			BinaryPath:       binary,
			TextLength:       len(text),
			Violations:       violations,
			CountsByRule:     map[string]int{},
			CountsByCategory: map[types.ViolationCategory]int{},
			Lang:             langName,
			LLMEffort:        string(eff),
			LLM:              llmRep,
			Readability:      read,
			Rules:            map[string]reportRule{},
			Config:           configFile,
		}
		for _, v := range violations {
			report.CountsByRule[v.RuleID]++
			if rule, ok := rules.ByID[v.RuleID]; ok {
				report.CountsByCategory[rule.Category]++
				report.Rules[v.RuleID] = reportRule{
					Name:        rule.Name,
					Category:    rule.Category,
					Tip:         rule.Tip,
					RewriteHint: rule.RewriteHint,
				}
			}
		}

		if outFormat == formatCompact {
			err = writeCompact(report, text)
		} else {
			err = writeJSON(report, *pretty)
		}
		if err != nil {
			return err
		}
		if strictFlag && len(report.CountsByRule) > 0 {
			return errViolationsFound
		}
		return nil
	}
	return cmd
}
