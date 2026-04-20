package lang

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// registry holds every registered Analyzer. Population happens through
// init() calls in each language package, so callers must blank-import those
// packages (or any package that transitively imports them) for the modes to
// be available.
var (
	registryMu  sync.RWMutex
	byName      = map[string]Analyzer{}
	byExtension = map[string]Analyzer{}
)

// Register wires an Analyzer into the registry under a.Name() and under
// every provided extension. Extensions are stored lower-cased and must
// include the leading dot ("." + "md"). Panics on duplicate name/extension;
// duplicates are a programmer error detectable at init time.
func Register(a Analyzer, extensions ...string) {
	registryMu.Lock()
	defer registryMu.Unlock()
	name := a.Name()
	if name == "" {
		panic("lang.Register: analyzer returned empty Name()")
	}
	if _, ok := byName[name]; ok {
		panic(fmt.Sprintf("lang.Register: duplicate analyzer name %q", name))
	}
	byName[name] = a
	for _, ext := range extensions {
		ext = strings.ToLower(ext)
		if ext == "" || !strings.HasPrefix(ext, ".") {
			panic(fmt.Sprintf("lang.Register(%s): extension %q must start with '.'", name, ext))
		}
		if prev, ok := byExtension[ext]; ok {
			panic(fmt.Sprintf("lang.Register: extension %q already claimed by %q", ext, prev.Name()))
		}
		byExtension[ext] = a
	}
}

// ByName returns the Analyzer registered under name. The special name "text"
// returns (nil, true) to signal "no masking, pass through as plain prose".
// The second result is false when the name is not recognised.
func ByName(name string) (Analyzer, bool) {
	name = strings.ToLower(name)
	if name == "text" {
		return nil, true
	}
	registryMu.RLock()
	defer registryMu.RUnlock()
	a, ok := byName[name]
	return a, ok
}

// ByExtension returns the Analyzer associated with a file extension (including
// the leading dot, case-insensitive). The second result is false when the
// extension has no registered analyzer — callers should treat that as "text".
func ByExtension(ext string) (Analyzer, bool) {
	ext = strings.ToLower(ext)
	registryMu.RLock()
	defer registryMu.RUnlock()
	a, ok := byExtension[ext]
	return a, ok
}

// Names returns every registered analyzer name plus "text", sorted. Used by
// the CLI flag help text and by tests that want to exercise every mode.
func Names() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]string, 0, len(byName)+1)
	out = append(out, "text")
	for name := range byName {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
