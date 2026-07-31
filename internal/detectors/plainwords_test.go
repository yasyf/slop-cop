package detectors

import (
	"sort"
	"strings"
	"testing"
)

func TestDetectPlainWord(t *testing.T) {
	runHits(t, "elevated-register", DetectPlainWord, []hitCase{
		{"accomplish", "The script accomplishes the migration in one pass.", true},
		{"cease", "The daemon ceased writing to the log.", true},
		{"conceal", "The wrapper conceals the retry loop.", true},
		{"encompass", "The base layer encompasses seven rules.", true},
		{"entail", "The change entails a schema migration.", true},
		{"possess", "The token possesses no scopes.", true},
		{"preclude", "A locked file precludes an in-place edit.", true},
		{"procure", "We procured a second runner.", true},
		{"assist", "The wizard assists with the first run.", true},
		{"whilst", "The cache warms whilst the build runs.", true},
		{"amongst", "The load is split amongst four workers.", true},
		{"albeit", "The fix landed, albeit late.", true},
		{"myriad", "A myriad of flags control the pass.", true},
		{"plethora", "There was a plethora of options.", true},
		{"bespoke", "The team wrote a bespoke parser.", true},
		{"seamless", "The upgrade was seamless.", true},
		{"meticulous", "The review was meticulous.", true},
		{"paramount", "Latency is paramount here.", true},
		{"subsequently", "The job subsequently retried.", true},
		{"approximately", "The pass takes approximately ten seconds.", true},
		{"methodology", "The methodology is documented upstream.", true},
		{"verbiage", "Trim the verbiage in the header.", true},
		{"learnings", "The retro captured our learnings.", true},
		{"operationalize", "The team operationalized the checklist.", true},
		{"incentivize", "The bonus incentivizes shipping.", true},
		{"in order to", "Run the build in order to refresh the cache.", true},
		{"prior to", "Snapshot the tree prior to the rebase.", true},
		{"due to the fact that", "It failed due to the fact that the token expired.", true},
		{"in the event that", "In the event that the push is rejected, retry.", true},
		{"at this point in time", "At this point in time the flag is off.", true},
		{"a large number of", "A large number of files changed.", true},
		{"the majority of", "The majority of hits are false.", true},
		{"for the purpose of", "The branch exists for the purpose of testing.", true},
		{"with the exception of", "Every target builds with the exception of freebsd.", true},
		{"in excess of", "The run took in excess of ten minutes.", true},
		{"with regard to", "With regard to the schema, nothing changed.", true},
		{"in accordance with", "The output is in accordance with the spec.", true},
		{"in an effort to", "In an effort to cut noise, the rule was dropped.", true},
		{"each and every", "Each and every push triggers a release.", true},
		{"first and foremost", "First and foremost, read the plan.", true},
		{"exact same", "The output is the exact same.", true},
		{"revert back", "Revert back to the previous tag.", true},
		{"end result", "The end result is a single binary.", true},
		{"past history", "The past history of the branch is intact.", true},
		{"various different", "Various different modes are supported.", true},
		{"not uncommon", "A flake is not uncommon here.", true},

		{"plain prose", "The tool reads a file and prints JSON.", false},
		{"implement is ordinary software vocabulary", "We implement the interface in one file.", false},
		{"execute is ordinary", "The runner executes the command.", false},
		{"abort is ordinary", "A conflict aborts the rebase.", false},
		{"terminate is ordinary", "The process terminates on SIGINT.", false},
		{"invoke is ordinary", "The hook invokes the binary.", false},
		{"instantiate is ordinary", "The registry instantiates one analyzer.", false},
		{"deprecate is ordinary", "The flag is deprecated.", false},
		{"serialize is ordinary", "The report serializes to JSON.", false},
		{"reside is ordinary", "The manifests reside under .claude-plugin.", false},
		{"personnel is ordinary", "The personnel roster is elsewhere.", false},
		{"functionality has no single replacement", "The functionality is unchanged.", false},
	})
}

// The upstream elevatedRegister list owns these stems, so this detector must
// leave their inflections alone (decision D16).
func TestDetectPlainWordSkipsUpstreamInflections(t *testing.T) {
	for _, s := range []string{
		"The script utilizes the cache.",
		"The job commenced at noon.",
		"The wrapper facilitated the handoff.",
		"We endeavoured to ship it.",
		"The run demonstrated the bug.",
		"We ascertained the cause.",
		"A crafted ref reproduces it.",
		"The change ameliorated the latency.",
	} {
		if got := DetectPlainWord(s); len(got) != 0 {
			t.Fatalf("DetectPlainWord(%q) = %+v, want none (upstream owns the stem)", s, got)
		}
	}
}

// Measured over 862K words of real markdown, these outranked every genuine
// tell combined: they are base engineering vocabulary, not elevated register.
// "with respect to" and "in relation to" went with them because both name a
// variable in mathematical and statistical writing, where "about" is wrong
// rather than plainer, and "terminology" because it is a singular mass noun:
// applying "terms" literally yields "the terms is inconsistent".
func TestDetectPlainWordSkipsOrdinaryVocabulary(t *testing.T) {
	for _, s := range []string{
		"The role has insufficient permissions for that bucket.",
		"Additional configuration options live in the manifest.",
		"Allocate a sufficient buffer before the call.",
		"The buffer is sufficiently large for the payload.",
		"Differentiate the loss with respect to the weights before the update step.",
		"Measure the variance in relation to the sample mean.",
		"The terminology is inconsistent across the codebase.",
		"Our terminology has drifted between the two teams.",
	} {
		if got := DetectPlainWord(s); len(got) != 0 {
			t.Fatalf("DetectPlainWord(%q) = %+v, want none", s, got)
		}
	}
}

func TestDetectPlainWordSkipsExcludedEntries(t *testing.T) {
	for _, s := range []string{
		"The fact that it works is enough.",
		"The result is not unlike the previous one.",
		"Licensor hereby grants the rights described thereof.",
		"Notwithstanding the above, the files are furnished as is.",
		"We provide assistance on request.",
	} {
		if got := DetectPlainWord(s); len(got) != 0 {
			t.Fatalf("DetectPlainWord(%q) = %+v, want none", s, got)
		}
	}
}

func TestDetectPlainWordHyphenGuard(t *testing.T) {
	if got := DetectPlainWord("AI-assisted workflows are common."); len(got) != 0 {
		t.Fatalf("hyphenated compound fired: %+v", got)
	}
	if got := DetectPlainWord("The wizard assisted the user."); len(got) != 1 {
		t.Fatalf("bare word did not fire: %+v", got)
	}
}

func TestDetectPlainWordWrappedPhrase(t *testing.T) {
	cases := []struct {
		name string
		text string
		want bool
	}{
		{"wrapped on LF", "Run the build in order\nto refresh the cache.", true},
		{"wrapped on CRLF", "Run the build in order\r\nto refresh the cache.", true},
		{"split by a paragraph break", "Run the build in order\n\nto refresh the cache.", false},
		{"no word boundary", "Run the build inorder to refresh the cache.", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := hasRule(DetectPlainWord(c.text), "elevated-register"); got != c.want {
				t.Fatalf("fires=%v want=%v for %q", got, c.want, c.text)
			}
		})
	}
}

func TestDetectPlainWordAlwaysSuggests(t *testing.T) {
	for _, s := range append(append([]plainSub{}, plainWordSubs...), plainPhraseSubs...) {
		v := DetectPlainWord("The " + s.from + " case.")
		if len(v) == 0 {
			t.Fatalf("%q did not fire", s.from)
		}
		if v[0].SuggestedChange != s.to {
			t.Fatalf("%q suggested %q, want %q", s.from, v[0].SuggestedChange, s.to)
		}
	}
}

func TestPlainSubListShape(t *testing.T) {
	if len(plainWordSubs) != 103 {
		t.Errorf("plainWordSubs = %d entries, want 103", len(plainWordSubs))
	}
	if len(plainPhraseSubs) != 43 {
		t.Errorf("plainPhraseSubs = %d entries, want 43", len(plainPhraseSubs))
	}
	seen := map[string]bool{}
	for _, s := range append(append([]plainSub{}, plainWordSubs...), plainPhraseSubs...) {
		if s.from != strings.ToLower(s.from) {
			t.Errorf("%q is not lowercase", s.from)
		}
		if s.to == "" || s.to == s.from {
			t.Errorf("%q has no useful replacement (%q)", s.from, s.to)
		}
		if seen[s.from] {
			t.Errorf("%q is duplicated", s.from)
		}
		seen[s.from] = true
	}
	for _, s := range plainWordSubs {
		if strings.ContainsAny(s.from, " \t") {
			t.Errorf("%q is a phrase, not a word", s.from)
		}
	}
	for _, s := range plainPhraseSubs {
		if !strings.Contains(s.from, " ") {
			t.Errorf("%q is a word, not a phrase", s.from)
		}
	}
}

// upstreamLists is every word list word_patterns.go detects against. A plain-word
// entry that appears in one of them would double-fire or fight D16.
func upstreamLists() map[string][]string {
	return map[string][]string{
		"intensifiers":               intensifiers,
		"elevatedRegister":           elevatedRegister,
		"fillerAdverbs":              fillerAdverbs,
		"metaphorCrutches":           metaphorCrutches,
		"falseConclusionPhrases":     falseConclusionPhrases,
		"connectorWords":             connectorWords,
		"unnecessaryContrastPhrases": unnecessaryContrastPhrases,
		"hedgeWords":                 hedgeWords,
		"commaQualifiers":            commaQualifiers,
		"heresTheKickerPhrases":      heresTheKickerPhrases,
		"pedagogicalPhrases":         pedagogicalPhrases,
		"vagueAttributionPhrases":    vagueAttributionPhrases,
	}
}

func TestPlainSubsDisjointFromUpstream(t *testing.T) {
	lists := upstreamLists()
	if len(lists) != 12 {
		t.Fatalf("upstreamLists() covers %d lists, want 12", len(lists))
	}
	owner := map[string]string{}
	for name, list := range lists {
		for _, w := range list {
			owner[strings.ToLower(w)] = name
		}
	}
	for _, s := range append(append([]plainSub{}, plainWordSubs...), plainPhraseSubs...) {
		if name, ok := owner[s.from]; ok {
			t.Errorf("%q already belongs to %s; drop it from the plain-word list", s.from, name)
		}
	}
}

// synthetic joins every entry into one document, one entry per sentence.
func synthetic() string {
	var b strings.Builder
	for _, s := range append(append([]plainSub{}, plainWordSubs...), plainPhraseSubs...) {
		b.WriteString("The " + s.from + " case is here. ")
	}
	return b.String()
}

func TestPlainWordsNeverOverlap(t *testing.T) {
	doc := synthetic()
	got := append(DetectPlainWord(doc), DetectElevatedRegister(doc)...)
	sort.Slice(got, func(i, j int) bool { return got[i].StartIndex < got[j].StartIndex })
	for i := 1; i < len(got); i++ {
		if got[i].StartIndex < got[i-1].EndIndex {
			t.Fatalf("overlapping elevated-register spans: %q [%d,%d) and %q [%d,%d)",
				got[i-1].MatchedText, got[i-1].StartIndex, got[i-1].EndIndex,
				got[i].MatchedText, got[i].StartIndex, got[i].EndIndex)
		}
	}
	if len(got) != len(plainWordSubs)+len(plainPhraseSubs) {
		t.Fatalf("got %d hits over %d entries", len(got), len(plainWordSubs)+len(plainPhraseSubs))
	}
}

// The plain-word list extends the elevated-register vocabulary from a sibling
// file: DetectElevatedRegister and its list stay byte-untouched, so a document
// made only of new entries is invisible to it.
func TestDetectElevatedRegisterUnchanged(t *testing.T) {
	if got := DetectElevatedRegister(synthetic()); len(got) != 0 {
		t.Fatalf("DetectElevatedRegister fired on plain-word-only text: %+v", got)
	}
	fixture := "We should utilize this tool at this juncture, moving forward."
	before := DetectElevatedRegister(fixture)
	if len(before) != 3 {
		t.Fatalf("upstream detector returned %d hits on its own fixture, want 3", len(before))
	}
	for _, v := range before {
		if v.SuggestedChange != "" {
			t.Fatalf("upstream hit gained a suggestion: %+v", v)
		}
	}
}
