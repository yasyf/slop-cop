package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/yasyf/slop-cop/internal/rules"
	"github.com/yasyf/slop-cop/internal/types"
)

// configName is the repo-local config file `check` walks up to find.
const configName = ".slopcop.toml"

// config is a decoded .slopcop.toml. Disable removes rules; EnableOnly
// restricts the run to the rules it names; Overrides layer path-scoped
// versions of both on top, in file order.
type config struct {
	Disable    []string         `toml:"disable"`
	EnableOnly []string         `toml:"enable_only"`
	Overrides  []configOverride `toml:"overrides"`
}

// configOverride is one [[overrides]] block: a path-glob selector plus the
// same two rule lists.
type configOverride struct {
	Paths      []string `toml:"paths"`
	Disable    []string `toml:"disable"`
	EnableOnly []string `toml:"enable_only"`
}

// ruleFilter decides which rule IDs reach the report and the LLM prompt. The
// zero value keeps every rule.
type ruleFilter struct {
	disabled   map[string]bool
	enableOnly map[string]bool
}

func (f ruleFilter) empty() bool { return f.disabled == nil && f.enableOnly == nil }

func (f ruleFilter) allows(ruleID string) bool {
	if f.enableOnly != nil && !f.enableOnly[ruleID] {
		return false
	}
	return !f.disabled[ruleID]
}

// keep drops the violations whose rule the filter excludes.
func (f ruleFilter) keep(vs []types.Violation) []types.Violation {
	if f.empty() {
		return vs
	}
	out := vs[:0]
	for _, v := range vs {
		if f.allows(v.RuleID) {
			out = append(out, v)
		}
	}
	return out
}

// catalogue narrows a rule catalogue to what the filter allows, so a
// suppressed rule never reaches the LLM prompt.
func (f ruleFilter) catalogue(rs []types.ViolationRule) []types.ViolationRule {
	if f.empty() {
		return rs
	}
	out := make([]types.ViolationRule, 0, len(rs))
	for _, r := range rs {
		if f.allows(r.ID) {
			out = append(out, r)
		}
	}
	return out
}

// resolveRuleFilter compiles the config governing this run into a filter for
// the input at path, and returns the file it came from ("" when none applied).
// configFlag overrides discovery and must exist; a discovered file's absence
// is not an error.
func resolveRuleFilter(configFlag, path string) (ruleFilter, string, error) {
	file := configFlag
	if file == "" {
		found, err := findConfig(path)
		if err != nil {
			return ruleFilter{}, "", err
		}
		if found == "" {
			return ruleFilter{}, "", nil
		}
		file = found
	}

	cfg, err := loadConfig(file)
	if err != nil {
		return ruleFilter{}, "", err
	}
	filter, err := cfg.filterFor(file, path)
	if err != nil {
		return ruleFilter{}, "", err
	}
	return filter, file, nil
}

// findConfig walks up from the input's directory, or from the working
// directory when the input is stdin.
func findConfig(path string) (string, error) {
	start := "."
	if path != "" && path != "-" {
		start = filepath.Dir(path)
	}
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolving %s: %w", start, err)
	}
	for {
		candidate := filepath.Join(dir, configName)
		switch _, err := os.Stat(candidate); {
		case err == nil:
			return candidate, nil
		case !errors.Is(err, fs.ErrNotExist):
			return "", fmt.Errorf("reading %s: %w", candidate, err)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}

func loadConfig(file string) (config, error) {
	var cfg config
	md, err := toml.DecodeFile(file, &cfg)
	if err != nil {
		return config{}, fmt.Errorf("reading %s: %w", file, err)
	}
	undecoded := md.Undecoded()
	if len(undecoded) == 0 {
		return cfg, nil
	}
	keys := make([]string, 0, len(undecoded))
	for _, k := range undecoded {
		keys = append(keys, k.String())
	}
	return config{}, usageError{err: fmt.Errorf("%s: unknown key %s (want disable, enable_only, or an [[overrides]] block)", file, strings.Join(keys, ", "))}
}

// filterFor layers the top-level lists and every matching override into one
// filter. Globs match the input path relative to the config file's directory.
func (c config) filterFor(file, path string) (ruleFilter, error) {
	disable := append([]string(nil), c.Disable...)
	enableOnly := c.EnableOnly
	enableOnlySet := c.EnableOnly != nil

	rel, err := relativeToConfig(file, path)
	if err != nil {
		return ruleFilter{}, err
	}
	for i, o := range c.Overrides {
		if len(o.Paths) == 0 {
			return ruleFilter{}, usageError{err: fmt.Errorf("%s: [[overrides]] block %d has no paths", file, i+1)}
		}
		matched, err := matchAnyGlob(o.Paths, rel)
		if err != nil {
			return ruleFilter{}, usageError{err: fmt.Errorf("%s: [[overrides]] block %d: %w", file, i+1, err)}
		}
		if !matched {
			continue
		}
		disable = append(disable, o.Disable...)
		if o.EnableOnly != nil {
			enableOnly, enableOnlySet = o.EnableOnly, true
		}
	}

	if err := validateRuleIDs(file, disable, enableOnly); err != nil {
		return ruleFilter{}, err
	}
	filter := ruleFilter{}
	if len(disable) > 0 {
		filter.disabled = idSet(disable)
	}
	if enableOnlySet {
		filter.enableOnly = idSet(enableOnly)
	}
	return filter, nil
}

// relativeToConfig expresses path relative to the config file's directory.
// Stdin has no path; an input outside that directory keeps its own spelling.
func relativeToConfig(file, path string) (string, error) {
	if path == "" || path == "-" {
		return "", nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolving %s: %w", path, err)
	}
	rel, err := filepath.Rel(filepath.Dir(file), abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(path), nil //nolint:nilerr // an input outside the config's tree still matches on its own spelling.
	}
	return filepath.ToSlash(rel), nil
}

func matchAnyGlob(patterns []string, rel string) (bool, error) {
	if rel == "" {
		return false, nil
	}
	for _, p := range patterns {
		re, err := compileGlob(p)
		if err != nil {
			return false, err
		}
		if re.MatchString(rel) {
			return true, nil
		}
		if !strings.Contains(p, "/") && re.MatchString(filepath.Base(rel)) {
			return true, nil
		}
	}
	return false, nil
}

// compileGlob translates a path glob into an anchored regexp. `**` crosses
// directory separators, `*` and `?` do not; everything else is literal.
func compileGlob(pattern string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString(`\A`)
	for i := 0; i < len(pattern); i++ {
		switch c := pattern[i]; c {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				i++
				if i+1 < len(pattern) && pattern[i+1] == '/' {
					i++
					b.WriteString(`(?:[^/]+/)*`)
					continue
				}
				b.WriteString(`.*`)
				continue
			}
			b.WriteString(`[^/]*`)
		case '?':
			b.WriteString(`[^/]`)
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	b.WriteString(`\z`)
	re, err := regexp.Compile(b.String())
	if err != nil {
		return nil, fmt.Errorf("invalid path glob %q", pattern)
	}
	return re, nil
}

func validateRuleIDs(file string, lists ...[]string) error {
	var unknown []string
	seen := map[string]bool{}
	for _, list := range lists {
		for _, id := range list {
			if _, ok := rules.ByID[id]; !ok && !seen[id] {
				seen[id] = true
				unknown = append(unknown, id)
			}
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	sort.Strings(unknown)
	return usageError{err: fmt.Errorf("%s: unknown rule %s (run `slop-cop rules` for the catalogue)", file, strings.Join(unknown, ", "))}
}

func idSet(ids []string) map[string]bool {
	out := make(map[string]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}
	return out
}
