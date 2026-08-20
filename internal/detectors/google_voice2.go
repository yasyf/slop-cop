package detectors

import (
	"regexp"
	"sort"
	"strings"

	"github.com/yasyf/slop-cop/internal/types"
)

const (
	voice2ruleThatWhich = "that-which-clause"
	voice2ruleTrivial   = "trivializing-difficulty"
	voice2window        = 64
)

func voice2set(words ...string) map[string]bool {
	m := make(map[string]bool, len(words))
	for _, w := range words {
		m[w] = true
	}
	return m
}

func voice2norm(w string) string {
	return strings.ToLower(strings.ReplaceAll(w, "’", "'"))
}

var (
	voice2rePrevWord = regexp.MustCompile(`([A-Za-z][A-Za-z'\x{2019}-]*)[ \t]*(?:\r?\n[ \t]*)?$`)
	voice2reNextWord = regexp.MustCompile(`^[ \t]*(?:\r?\n[ \t]*)?([A-Za-z][A-Za-z'\x{2019}-]*)`)
)

func voice2prevWord(text string, start int) string {
	from := start - voice2window
	if from < 0 {
		from = 0
	}
	m := voice2rePrevWord.FindStringSubmatch(text[from:start])
	if m == nil {
		return ""
	}
	return m[1]
}

func voice2nextWord(text string, end int) string {
	to := end + voice2window
	if to > len(text) {
		to = len(text)
	}
	m := voice2reNextWord.FindStringSubmatch(text[end:to])
	if m == nil {
		return ""
	}
	return m[1]
}

func voice2dedupe(hits []types.Violation) []types.Violation {
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].StartIndex != hits[j].StartIndex {
			return hits[i].StartIndex < hits[j].StartIndex
		}
		return hits[i].EndIndex > hits[j].EndIndex
	})
	out := make([]types.Violation, 0, len(hits))
	covered := -1
	for _, h := range hits {
		if h.StartIndex < covered {
			continue
		}
		out = append(out, h)
		if h.EndIndex > covered {
			covered = h.EndIndex
		}
	}
	return out
}

var (
	voice2reCommaThat = regexp.MustCompile(`,` + phraseSep + `(that)` + phraseSep +
		`([a-z][a-z'\x{2019}-]*)`)
	voice2reBareWhich = regexp.MustCompile(`([A-Za-z][A-Za-z'\x{2019}-]*)` + phraseSep +
		`(which)` + phraseSep + `([a-z][a-z'\x{2019}-]*)`)
)

var voice2thatAux = voice2set(
	"is", "are", "was", "were", "has", "have", "had", "does", "do", "did",
	"can", "cannot", "could", "will", "shall", "should", "would", "may",
	"might", "must", "isn't", "aren't", "wasn't", "weren't", "hasn't",
	"haven't", "hadn't", "doesn't", "don't", "didn't", "can't", "won't",
	"wouldn't", "shouldn't", "couldn't", "mustn't",
)

var voice2thatNonVerbal = voice2set(
	"series", "news", "means", "species", "gas", "lens", "its", "alias",
	"canvas", "atlas", "bias", "yours", "theirs", "ours", "hers",
)

var voice2thatEdNouns = voice2set(
	"speed", "feed", "seed", "need", "deed", "breed", "bed", "shed", "embed", "creed",
)

// "that is" carries a parenthetical sense (", that is,"), a cleft sense
// (", that is why"), and a predicate nominal whose subject is the demonstrative
// pronoun (", that is a findability bug"). The rule is about none of them.
var voice2thatIsDeictic = voice2set(
	"why", "how", "what", "when", "where", "because",
	"a", "an", "the", "this", "that", "these", "those", "one", "another",
	"my", "your", "our", "their", "its", "his", "her",
	"no", "not", "just", "only", "something", "nothing",
)

func voice2looksVerbal(w string) bool {
	if voice2thatNonVerbal[w] {
		return false
	}
	if strings.HasSuffix(w, "ss") || strings.HasSuffix(w, "us") || strings.HasSuffix(w, "is") {
		return false
	}
	if strings.HasSuffix(w, "s") {
		return true
	}
	return strings.HasSuffix(w, "ed") && !voice2thatEdNouns[w]
}

func voice2isVerbal(w string) bool { return voice2thatAux[w] || voice2looksVerbal(w) }

func voice2isRelativeFollower(text, follower string, end int) bool {
	if follower == "is" {
		if end < len(text) && strings.ContainsRune(",:;", rune(text[end])) {
			return false
		}
		return !voice2thatIsDeictic[voice2norm(voice2nextWord(text, end))]
	}
	return voice2isVerbal(follower)
}

var voice2reClauseCloses = regexp.MustCompile(`^[^,.!?\n]*,`)

// A `that` clause set off on both sides is the nonrestrictive punctuation the
// rule is about. A trailing one is more often a comma splice whose `that` is a
// demonstrative pronoun ("re-run ship, that cuts a new commit").
func voice2clauseCloses(text string, end int) bool {
	to := end + 200
	if to > len(text) {
		to = len(text)
	}
	return voice2reClauseCloses.MatchString(text[end:to])
}

var voice2whichPrevDrop = voice2set(
	"about", "above", "across", "after", "against", "along", "alongside",
	"among", "amongst", "around", "at", "before", "behind", "below",
	"beneath", "beside", "between", "beyond", "by", "concerning", "despite",
	"down", "during", "except", "for", "from", "in", "inside", "into",
	"near", "of", "off", "on", "onto", "opposite", "out", "outside", "over",
	"past", "per", "regarding", "since", "through", "throughout", "till",
	"to", "toward", "towards", "under", "underneath", "until", "unto", "up",
	"upon", "via", "with", "within", "without",

	"and", "or", "but", "nor", "so", "yet", "that", "than", "then", "no",
	"not", "matter", "matters", "mattered", "regardless", "depending",

	"is", "are", "was", "were", "am", "be", "been", "being",
	"it", "them", "him", "her", "us", "me", "user", "users", "host", "hosts",

	"know", "knows", "knew", "known", "knowing",
	"see", "sees", "saw", "seen", "seeing",
	"determine", "determines", "determined", "determining",
	"specify", "specifies", "specified", "specifying",
	"decide", "decides", "decided", "deciding",
	"depend", "depends", "depended",
	"choose", "chooses", "chose", "chosen", "choosing",
	"indicate", "indicates", "indicated", "indicating",
	"show", "shows", "showed", "shown", "showing",
	"control", "controls", "controlled", "controlling",
	"find", "finds", "found", "finding",
	"identify", "identifies", "identified", "identifying",
	"track", "tracks", "tracked", "tracking",
	"record", "records", "recorded", "recording",
	"tell", "tells", "told", "telling",
	"check", "checks", "checked", "checking",
	"note", "notes", "noted", "noting",
	"notice", "notices", "noticed",
	"observe", "observes", "observed",
	"learn", "learns", "learned", "learnt",
	"remember", "remembers", "remembered",
	"ask", "asks", "asked",
	"discover", "discovers", "discovered",
	"verify", "verifies", "verified",
	"confirm", "confirms", "confirmed",
	"report", "reports", "reported",
	"list", "lists", "listed",
	"display", "displays", "displayed",
	"log", "logs", "logged",
	"count", "counts", "counted",
	"select", "selects", "selected",
	"pick", "picks", "picked",
	"evaluate", "evaluates", "evaluated",
	"compute", "computes", "computed",
	"calculate", "calculates", "calculated",
	"understand", "understands", "understood",
	"examine", "examines", "examined",
	"inspect", "inspects", "inspected",
	"review", "reviews", "reviewed",
	"wonder", "wonders", "wondered",
	"describe", "describes", "described",
	"document", "documents", "documented",
	"explain", "explains", "explained",
	"define", "defines", "defined",
	"dictate", "dictates", "dictated",
	"govern", "governs", "governed",
	"reflect", "reflects", "reflected",
	"predict", "predicts", "predicted",
	"estimate", "estimates", "estimated",
	"detect", "detects", "detected",
	"state", "states", "stated",
	"signal", "signals", "signaled",
	"guess", "guesses", "guessed",
	"illustrate", "illustrates", "illustrated",
	"clarify", "clarifies", "clarified",
	"judge", "judges", "judged",
	"say", "says", "said", "saying",
	"care", "cares", "cared",
	"restrict", "restricts", "restricted", "restricting",
	"drive", "drives", "drove", "driven", "driving",
	"reporting", "announce", "announces", "announced", "announcing",
	"change", "changes", "changed", "changing",
	"enrich", "enriches", "enriched",
	"establish", "establishes", "established", "establishing",
	"diagnose", "diagnoses", "diagnosed",
	"filter", "filters", "filtered", "filtering",
	"customize", "customizes", "customized", "customizing",
	"affect", "affects", "affected", "affecting",
	"name", "names", "named", "naming",
)

var voice2whichNextDrop = voice2set("one", "ones", "of", "ever")

var voice2lyNouns = voice2set("assembly", "family", "supply", "reply", "anomaly", "ally")

// A candidate antecedent that opens its clause is an imperative verb, not the
// noun a relative clause attaches to: "Filter which requests get spans".
func voice2opensClause(text string, start int) bool {
	i := start - 1
	for i >= 0 && (text[i] == ' ' || text[i] == '\t') {
		i--
	}
	if i < 0 {
		return true
	}
	if text[i] == '\n' || text[i] == '\r' {
		return voice2lineOpensClause(text, i)
	}
	return strings.IndexByte("([:;.!?|>*#-", text[i]) >= 0
}

// A line break only opens a clause when the line before it ended one; a
// hard-wrapped sentence carries its clause across the break.
func voice2lineOpensClause(text string, nl int) bool {
	i := nl
	for i >= 0 && (text[i] == '\n' || text[i] == '\r' || text[i] == ' ' || text[i] == '\t') {
		i--
	}
	if i < 0 {
		return true
	}
	return strings.IndexByte(".!?:;|", text[i]) >= 0
}

func voice2isAntecedent(text, prev string, start int) bool {
	if voice2whichPrevDrop[prev] || voice2opensClause(text, start) {
		return false
	}
	return !strings.HasSuffix(prev, "ly") || voice2lyNouns[prev]
}

// DetectGoogleThatWhichClause reports a comma before a restrictive `that` and a
// restrictive `which` introduced without one.
func DetectGoogleThatWhichClause(text string) []types.Violation {
	var out []types.Violation

	for _, m := range voice2reCommaThat.FindAllStringSubmatchIndex(text, -1) {
		follower := voice2norm(text[m[4]:m[5]])
		if !voice2isRelativeFollower(text, follower, m[5]) || !voice2clauseCloses(text, m[3]) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          voice2ruleThatWhich,
			StartIndex:      m[0],
			EndIndex:        m[3],
			MatchedText:     text[m[0]:m[3]],
			Explanation:     "a clause introduced by `that` identifies its noun and takes no comma",
			SuggestedChange: ", which",
		})
	}

	for _, m := range voice2reBareWhich.FindAllStringSubmatchIndex(text, -1) {
		if !voice2isAntecedent(text, voice2norm(text[m[2]:m[3]]), m[2]) {
			continue
		}
		next := voice2norm(text[m[6]:m[7]])
		if voice2whichNextDrop[next] || !voice2isVerbal(next) {
			continue
		}
		// "which claims are high-stakes": a plural noun carrying its own verb
		// is an embedded question, not a relative clause.
		if !voice2thatAux[next] && voice2thatAux[voice2norm(voice2nextWord(text, m[7]))] {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          voice2ruleThatWhich,
			StartIndex:      m[4],
			EndIndex:        m[5],
			MatchedText:     text[m[4]:m[5]],
			Explanation:     "an aside takes `, which`; a clause the sentence needs takes `that`",
			SuggestedChange: "that",
		})
	}

	return voice2dedupe(out)
}

const voice2trivialWhy = "rates the reader's effort or the step's duration"

var (
	voice2reTrivialWord = regexp.MustCompile(`(?i)\b(?:simply|simple|merely|easily|easy|` +
		`effortlessly|quickly|quick|obviously|clearly|trivially|straightforward|` +
		`painless|sexy)\b`)
	voice2reTrivialItsSimple = regexp.MustCompile(`(?i)\bit['\x{2019}]?s\s+` +
		`(?:that\s+simple|simple|easy|quick|trivial)\b`)
	voice2reTrivialAllYou = regexp.MustCompile(`(?i)\ball\s+you\s+` +
		`(?:have\s+to\s+do|need\s+to\s+do|need)\b`)
	voice2reTrivialJust = regexp.MustCompile(`(?i)\bjust\s+(?:to|a|an|the|one|now|` +
		`run|click|call|add|type|set|open|enter|use|copy|paste|wait|works?|needs?|wants?|` +
		`skips?|reads?|writes?|returns?|creates?|deletes?)\b`)
	voice2reTrivialOfCourse = regexp.MustCompile(`(?i)\bof\s+course\b`)
)

var (
	voice2simpleProperNouns = voice2set(
		"Storage", "Notification", "Queue", "Object", "Mail", "Network", "Authentication",
	)
	voice2quickCompounds = voice2set("reference", "start", "starts")
	voice2justPrevDrop   = voice2set("or", "than", "not")
)

func voice2trivialSuppressed(text string, start, end int) bool {
	word := text[start:end]
	switch strings.ToLower(word) {
	case "quick":
		return voice2quickCompounds[voice2norm(voice2nextWord(text, end))]
	case "simple":
		return word[0] == 'S' && voice2simpleProperNouns[voice2nextWord(text, end)]
	case "clearly":
		// Only the sentence adverb rates the reader; "label the axes clearly"
		// is a manner adverb.
		return end >= len(text) || text[end] != ','
	}
	return false
}

// DetectGoogleTrivializingDifficulty reports words that rate a step's
// difficulty or duration for the reader.
func DetectGoogleTrivializingDifficulty(text string) []types.Violation {
	var out []types.Violation

	for _, idx := range voice2reTrivialWord.FindAllStringIndex(text, -1) {
		if voice2trivialSuppressed(text, idx[0], idx[1]) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:      voice2ruleTrivial,
			StartIndex:  idx[0],
			EndIndex:    idx[1],
			MatchedText: text[idx[0]:idx[1]],
			Explanation: voice2trivialWhy,
		})
	}

	for _, idx := range voice2reTrivialJust.FindAllStringIndex(text, -1) {
		if voice2justPrevDrop[voice2norm(voice2prevWord(text, idx[0]))] {
			continue
		}
		out = append(out, types.Violation{
			RuleID:      voice2ruleTrivial,
			StartIndex:  idx[0],
			EndIndex:    idx[1],
			MatchedText: text[idx[0]:idx[1]],
			Explanation: voice2trivialWhy,
		})
	}

	for _, re := range []*regexp.Regexp{
		voice2reTrivialItsSimple,
		voice2reTrivialAllYou,
		voice2reTrivialOfCourse,
	} {
		for _, v := range findAll(text, re, voice2ruleTrivial) {
			v.Explanation = voice2trivialWhy
			out = append(out, v)
		}
	}

	return voice2dedupe(out)
}
