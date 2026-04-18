package detectors

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/yasyf/slop-cop/internal/types"
)

// ── Word lists (ported verbatim from src/detectors/wordPatterns.ts) ─────────

var intensifiers = []string{
	"crucial", "vital", "robust", "comprehensive", "fundamental",
	"arguably", "straightforward", "noteworthy", "realm", "landscape",
	"leverage", "delve", "tapestry", "multifaceted", "nuanced", "pivotal",
	"unprecedented", "navigate", "foster", "underscore", "resonate",
	"embark", "streamline", "spearhead", "paradigm", "synergy",
	"holistic", "transformative", "cutting-edge", "innovative", "dynamic",
	"harness",
}

// Only the keys of ELEVATED_REGISTER are used for detection.
var elevatedRegister = []string{
	"utilize", "utilise", "utilization",
	"commence", "commencement",
	"facilitate",
	"endeavor", "endeavour",
	"demonstrate", "ascertain", "ameliorate", "elucidate", "promulgate",
	"cognizant", "craft",
	"pertaining to", "in regards to", "with regards to",
	"in the context of", "at this juncture",
	"going forward", "moving forward",
	"in terms of", "it is worth noting", "it should be noted",
	"one must consider", "in light of", "in the realm of",
}

var fillerAdverbs = []string{
	"importantly", "essentially", "fundamentally", "ultimately",
	"inherently", "particularly", "increasingly", "certainly",
	"undoubtedly", "obviously", "clearly", "simply", "basically",
	"quite", "rather", "very", "really", "truly", "genuinely",
	"quietly", "deeply", "remarkably",
}

// Entries with a literal '.' in place of '-'/'space' are expanded to
// `[- ]?` so that "game.changer" matches "game changer" and "game-changer",
// mirroring the JS source (`.replace(/\\\./g, '[- ]?')`).
var metaphorCrutches = []string{
	"double-edged sword", "tip of the iceberg", "north star",
	"building blocks", "elephant in the room", "perfect storm",
	"game.changer", "game changer", "low.hanging fruit", "low hanging fruit",
	"move the needle", "think outside the box", "at the end of the day",
	"paradigm shift", "silver bullet", "boiling the ocean",
	"drinking the kool.aid", "drinking the kool aid",
	"put it on the back burner", "circle back", "deep dive",
	"level up", "hit the ground running", "move fast and break things",
	"the devil is in the details", "on the same page",
	"reinvent the wheel", "touch base", "bandwidth",
	"bleeding edge", "best of breed", "boil down",
}

var falseConclusionPhrases = []string{
	"in conclusion", "to conclude", "in summary", "to summarize",
	"to sum up", "in closing", "overall,", "all in all",
	"at the end of the day", "when all is said and done",
	"taking everything into account", "taking everything into consideration",
	"all things considered", "moving forward", "going forward",
}

var connectorWords = []string{
	"furthermore", "moreover", "additionally", "however", "nevertheless",
	"nonetheless", "consequently", "therefore", "thus", "hence",
	"in addition", "as a result", "for instance", "for example",
	"in contrast", "on the other hand", "on the contrary", "that said",
	"having said that", "with that in mind", "it follows that",
	"interestingly", "notably", "significantly",
}

var unnecessaryContrastPhrases = []string{
	"whereas", "as opposed to", "unlike", "in contrast to",
	"contrary to", "conversely",
}

var hedgeWords = []string{
	"perhaps", "arguably", "seemingly", "apparently", "ostensibly",
	"possibly", "potentially", "conceivably", "presumably", "supposedly",
	"it could be argued", "it might be", "it may be", "it seems",
	"it appears", "one might", "some would say", "in some ways",
	"to some extent", "in a sense", "sort of",
	"is kind of", "are kind of", "was kind of", "were kind of",
	"feels kind of", "seems kind of", "sounds kind of", "looks kind of",
}

var commaQualifiers = []string{
	"of course", "to be fair", "it should be said", "needless to say",
	"in fairness", "admittedly", "to be sure", "it must be said",
	"after all", "as we know", "as everyone knows",
}

var heresTheKickerPhrases = []string{
	"here's the kicker",
	"here's the thing",
	"here's where it gets interesting",
	"here's what most people miss",
	"here's the real",
}

var pedagogicalPhrases = []string{
	"let's break this down",
	"let's unpack",
	"let's explore",
	"let's dive in",
	"let's examine",
	"think of it as",
	"think of it like",
	"think of this as",
}

var vagueAttributionPhrases = []string{
	"experts argue", "experts say", "experts suggest", "experts believe", "experts note",
	"industry analysts", "observers have noted", "observers have cited", "observers argue",
	"analysts note", "analysts suggest", "many experts", "several experts", "some experts",
	"according to experts", "studies show", "research suggests",
}

// ── Precompiled regexes for fixed patterns ─────────────────────────────────

var (
	reAlmostHedge        = regexp.MustCompile(`(?i)\balmost\s+(always|never|certainly|exclusively|entirely|completely|always|invariably|universally)\b`)
	reEraOpener          = regexp.MustCompile(`(?i)\bin\s+an?\s+era\s+(of|where|when|in\s+which)\b`)
	reImportantToNote    = regexp.MustCompile(`(?i)\b(it('s| is)\s+important\s+to\s+note|it('s| is)\s+worth\s+noting|notably|note\s+that|it\s+should\s+be\s+noted)\b`)
	reBroaderImplication = regexp.MustCompile(`(?i)\b(broader\s+implications?|wider\s+implications?|implications?\s+(for|of|on)\s+the\s+(broader|wider|larger))\b`)
	reDash               = regexp.MustCompile(`[\x{2014}\x{2013}]`)
	reColonElab          = regexp.MustCompile(`[^.!?\n]{5,50}:[^:\n]{20,}`)
	reParenQualifier     = regexp.MustCompile(`\([^)]{20,}\)`)
	reServesAs           = regexp.MustCompile(`(?i)\b(serves|stands|acts|functions)\s+as\b`)
	reImagineWorld       = regexp.MustCompile(`(?i)\bImagine\s+(a world|if you|what would|a future)`)
	reListicleTrench     = regexp.MustCompile(`(?i)(^|[.!?]\s+|\n\s*)the\s+(first|second|third|fourth|fifth)\b`)
	reBoldFirstBullet    = regexp.MustCompile(`(?m)^[ \t]*[-*\x{2022}][ \t]+\*\*[^*\n]+\*\*`)
	reUnicodeArrow       = regexp.MustCompile(`\x{2192}`)
	reDespite            = regexp.MustCompile(`(?i)\bDespite (these|its|the|their|all|such)\b[^.!?]{0,80}\b(challenge|obstacle|limitation|difficult|drawback|shortcoming)`)
	reConceptLabel       = regexp.MustCompile(`(?i)\b[a-z]+\s+(paradox|trap|creep|vacuum|inversion|chasm)\b`)
	reSuperficial        = regexp.MustCompile(`(?i),\s+(highlighting|underscoring|showcasing|reflecting|cementing|embodying|encapsulating)\s+(its|the|their|this)\s+(importance|role|significance|legacy|power|spirit|nature|value)\b`)
	reNumberedList       = regexp.MustCompile(`(?m)(?:^|\n)(\s*\d+[.)]\s+[^\n]+)(\n\s*\d+[.)]\s+[^\n]+){2,}`)
	reBulletList         = regexp.MustCompile(`(?m)(?:^|\n)(\s*[-*\x{2022}]\s+[^\n]+)(\n\s*[-*\x{2022}]\s+[^\n]+){2,}`)
	reNumberedItem       = regexp.MustCompile(`^\s*\d+[.)]\s`)
	reBulletItem         = regexp.MustCompile(`^\s*[-*\x{2022}]\s`)
	reCapitalGerund      = regexp.MustCompile(`^[A-Z][a-z]+ing\b`)
)

// ── Detectors ──────────────────────────────────────────────────────────────

func DetectOverusedIntensifiers(text string) []types.Violation {
	var out []types.Violation
	for _, w := range intensifiers {
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(w) + `s?\b`)
		out = append(out, findAll(text, re, "overused-intensifiers")...)
	}
	return out
}

func DetectElevatedRegister(text string) []types.Violation {
	var out []types.Violation
	for _, w := range elevatedRegister {
		re := regexp.MustCompile(`(?i)\b` + escapeForRegex(w) + `\b`)
		out = append(out, findAll(text, re, "elevated-register")...)
	}
	return out
}

func DetectFillerAdverbs(text string) []types.Violation {
	var out []types.Violation
	for _, w := range fillerAdverbs {
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(w) + `\b`)
		out = append(out, findAll(text, re, "filler-adverbs")...)
	}
	return out
}

func DetectAlmostHedge(text string) []types.Violation {
	return findAll(text, reAlmostHedge, "almost-hedge")
}

func DetectEraOpener(text string) []types.Violation {
	return findAll(text, reEraOpener, "era-opener")
}

func DetectMetaphorCrutch(text string) []types.Violation {
	var out []types.Violation
	for _, phrase := range metaphorCrutches {
		// Match the TS transform: escape metacharacters, then replace escaped
		// '.' with `[- ]?` so dotted entries double as hyphen/space variants.
		escaped := escapeForRegex(phrase)
		pattern := strings.ReplaceAll(escaped, `\.`, `[- ]?`)
		re := regexp.MustCompile(`(?i)\b` + pattern + `\b`)
		out = append(out, findAll(text, re, "metaphor-crutch")...)
	}
	return out
}

func DetectImportantToNote(text string) []types.Violation {
	return findAll(text, reImportantToNote, "important-to-note")
}

func DetectBroaderImplications(text string) []types.Violation {
	return findAll(text, reBroaderImplication, "broader-implications")
}

func DetectFalseConclusion(text string) []types.Violation {
	var out []types.Violation
	for _, phrase := range falseConclusionPhrases {
		re := regexp.MustCompile(`(?i)(^|[.!?]\s+|\n\s*)` + escapeForRegex(phrase) + `\b`)
		out = append(out, findAll(text, re, "false-conclusion")...)
	}
	return out
}

func DetectConnectorAddiction(text string) []types.Violation {
	var out []types.Violation
	for _, w := range connectorWords {
		re := regexp.MustCompile(`(?i)(^|\n\s*|[.!?]\s+)` + escapeForRegex(w) + `[,\s]`)
		out = append(out, findAll(text, re, "connector-addiction")...)
	}
	return out
}

func DetectUnnecessaryContrast(text string) []types.Violation {
	var out []types.Violation
	for _, phrase := range unnecessaryContrastPhrases {
		re := regexp.MustCompile(`(?i)\b` + escapeForRegex(phrase) + `\b`)
		out = append(out, findAll(text, re, "unnecessary-contrast")...)
	}
	return out
}

func DetectEmDashPivot(text string) []types.Violation {
	var out []types.Violation
	for _, idx := range reDash.FindAllStringIndex(text, -1) {
		lineStart := strings.LastIndexByte(text[:idx[0]], '\n') + 1
		lineEnd := len(text)
		if p := strings.IndexByte(text[idx[0]:], '\n'); p >= 0 {
			lineEnd = idx[0] + p
		}
		line := text[lineStart:lineEnd]
		stripped := strings.TrimSpace(reDash.ReplaceAllString(line, ""))
		if stripped == "" {
			continue
		}
		out = append(out, types.Violation{
			RuleID:      "em-dash-pivot",
			StartIndex:  idx[0],
			EndIndex:    idx[1],
			MatchedText: text[idx[0]:idx[1]],
		})
	}
	return out
}

// negationPivotNEG lists the negation forms used in the inline and two-clause
// variants. Unicode apostrophes handled the same way the TS source does.
const negationsClass = `not|don[\x{2019}']?t|doesn[\x{2019}']?t|isn[\x{2019}']?t|wasn[\x{2019}']?t|aren[\x{2019}']?t|do not|does not|is not|was not|never|no longer`
const twoSentenceNEG = `(?:doesn[\x{2019}']?t|isn[\x{2019}']?t|won[\x{2019}']?t|can[\x{2019}']?t|don[\x{2019}']?t|does\s+not|is\s+not|was\s+not|did\s+not|will\s+not)`

var (
	reNegationComma   = regexp.MustCompile(`(?i)\b(?:` + negationsClass + `)\b[^.!?\n]{3,80},?\s+but\b`)
	reNegationEmDash  = regexp.MustCompile(`(?i)\b(?:` + negationsClass + `)\b[^.!?\n\x{2014}\x{2013}]{3,60}[\x{2014}\x{2013}]\s*\w+`)
	// First clause of the two-sentence variant. The subject ([A-Z]\w*) is
	// captured so we can look for a matching second clause that reopens with
	// the same subject (RE2 has no backreferences, so we stitch by hand).
	reNegationTwoFirst = regexp.MustCompile(`([A-Z][\w\x{2019}']*)\s+` + twoSentenceNEG + `\b[^.!?\n]{5,120}[.!?]`)
)

func DetectNegationPivot(text string) []types.Violation {
	var out []types.Violation
	for _, re := range []*regexp.Regexp{reNegationComma, reNegationEmDash} {
		for _, idx := range re.FindAllStringIndex(text, -1) {
			out = append(out, types.Violation{
				RuleID:      "negation-pivot",
				StartIndex:  idx[0],
				EndIndex:    idx[1],
				MatchedText: text[idx[0]:idx[1]],
			})
		}
	}
	// Two-sentence variant: "<Subject> doesn't X. <Subject> Y."
	for _, m := range reNegationTwoFirst.FindAllStringSubmatchIndex(text, -1) {
		firstStart, firstEnd := m[0], m[1]
		subjStart, subjEnd := m[2], m[3]
		subject := text[subjStart:subjEnd]
		rest := text[firstEnd:]
		// Skip the inter-sentence whitespace ([ \t]+ in the original).
		sp := 0
		for sp < len(rest) && (rest[sp] == ' ' || rest[sp] == '\t') {
			sp++
		}
		if sp == 0 {
			continue
		}
		secondRe := regexp.MustCompile(regexp.QuoteMeta(subject) + `\b[^.!?\n]{5,120}[.!?]`)
		loc := secondRe.FindStringIndex(rest[sp:])
		if loc == nil || loc[0] != 0 {
			continue
		}
		totalEnd := firstEnd + sp + loc[1]
		out = append(out, types.Violation{
			RuleID:      "negation-pivot",
			StartIndex:  firstStart,
			EndIndex:    totalEnd,
			MatchedText: text[firstStart:totalEnd],
		})
	}
	return out
}

func DetectColonElaboration(text string) []types.Violation {
	var out []types.Violation
	for _, idx := range reColonElab.FindAllStringIndex(text, -1) {
		colonPos := strings.Index(text[idx[0]:idx[1]], ":")
		if colonPos < 0 {
			continue
		}
		pos := idx[0] + colonPos
		out = append(out, types.Violation{
			RuleID:      "colon-elaboration",
			StartIndex:  pos,
			EndIndex:    pos + 1,
			MatchedText: ":",
		})
	}
	return out
}

func DetectParentheticalQualifier(text string) []types.Violation {
	out := findAll(text, reParenQualifier, "parenthetical-qualifier")
	for _, phrase := range commaQualifiers {
		re := regexp.MustCompile(`(?i),\s*` + regexp.QuoteMeta(phrase) + `\s*,`)
		out = append(out, findAll(text, re, "parenthetical-qualifier")...)
	}
	return out
}

func DetectQuestionThenAnswer(text string) []types.Violation {
	var out []types.Violation
	sentenceAny := regexp.MustCompile(`[^.!?]*[.!?]+`)
	for _, p := range splitParagraphs(text) {
		type s struct {
			text  string
			start int
		}
		var sentences []s
		for _, idx := range sentenceAny.FindAllStringIndex(p.text, -1) {
			sentences = append(sentences, s{
				text:  p.text[idx[0]:idx[1]],
				start: p.start + idx[0],
			})
		}
		for i := 0; i < len(sentences)-1; i++ {
			cur := strings.TrimSpace(sentences[i].text)
			next := strings.TrimSpace(sentences[i+1].text)
			if strings.HasSuffix(cur, "?") && !strings.HasSuffix(next, "?") && len(next) <= 120 {
				start := sentences[i].start
				end := sentences[i+1].start + len(sentences[i+1].text)
				out = append(out, types.Violation{
					RuleID:      "question-then-answer",
					StartIndex:  start,
					EndIndex:    end,
					MatchedText: text[start:end],
				})
			}
		}
	}
	return out
}

var hedgeModals = []string{"might", "could", "may"}

func DetectHedgeStack(text string) []types.Violation {
	var out []types.Violation
	sentences := splitSentences(text)
	offset := 0
	for _, sent := range sentences {
		lower := strings.ToLower(sent)
		var found []string
		for _, h := range hedgeWords {
			if strings.Contains(lower, h) {
				found = append(found, h)
			}
		}
		for _, m := range hedgeModals {
			re := regexp.MustCompile(`\b` + m + `\b`)
			if re.MatchString(lower) {
				found = append(found, m)
			}
		}
		if len(found) >= 2 {
			limit := len(found)
			if limit > 4 {
				limit = 4
			}
			out = append(out, types.Violation{
				RuleID:      "hedge-stack",
				StartIndex:  offset,
				EndIndex:    offset + len(sent),
				MatchedText: sent,
				Explanation: "Contains " + itoa(len(found)) + " hedges: " + strings.Join(found[:limit], ", "),
			})
		}
		offset += len(sent)
	}
	return out
}

func DetectStaccatoBurst(text string) []types.Violation {
	var out []types.Violation
	for _, p := range splitParagraphs(text) {
		sentences := splitSentences(p.text)
		offsets := make([]int, len(sentences))
		off := 0
		for i, s := range sentences {
			offsets[i] = p.start + off
			off += len(s)
		}
		wordCount := func(s string) int {
			return len(strings.Fields(s))
		}
		i := 0
		for i < len(sentences) {
			if wordCount(sentences[i]) <= 8 {
				j := i + 1
				for j < len(sentences) && wordCount(sentences[j]) <= 8 {
					j++
				}
				if j-i >= 3 {
					start := offsets[i]
					end := offsets[j-1] + len(sentences[j-1])
					out = append(out, types.Violation{
						RuleID:      "staccato-burst",
						StartIndex:  start,
						EndIndex:    end,
						MatchedText: text[start:end],
						Explanation: itoa(j-i) + " consecutive short sentences",
					})
					i = j
					continue
				}
			}
			i++
		}
	}
	return out
}

func DetectListicleInstinct(text string) []types.Violation {
	var out []types.Violation
	magic := map[int]bool{3: true, 5: true, 7: true, 10: true}

	for _, idx := range reNumberedList.FindAllStringIndex(text, -1) {
		match := text[idx[0]:idx[1]]
		var items int
		for _, line := range strings.Split(strings.TrimSpace(match), "\n") {
			if reNumberedItem.MatchString(line) {
				items++
			}
		}
		if magic[items] {
			out = append(out, types.Violation{
				RuleID:      "listicle-instinct",
				StartIndex:  idx[0],
				EndIndex:    idx[1],
				MatchedText: match,
				Explanation: "Numbered list with exactly " + itoa(items) + " items",
			})
		}
	}
	for _, idx := range reBulletList.FindAllStringIndex(text, -1) {
		match := text[idx[0]:idx[1]]
		var items int
		for _, line := range strings.Split(strings.TrimSpace(match), "\n") {
			if reBulletItem.MatchString(line) {
				items++
			}
		}
		if magic[items] {
			out = append(out, types.Violation{
				RuleID:      "listicle-instinct",
				StartIndex:  idx[0],
				EndIndex:    idx[1],
				MatchedText: match,
				Explanation: "Bullet list with exactly " + itoa(items) + " items",
			})
		}
	}
	return out
}

func DetectServesAs(text string) []types.Violation {
	return findAll(text, reServesAs, "serves-as")
}

func DetectNegationCountdown(text string) []types.Violation {
	var out []types.Violation
	sentences := splitSentences(text)
	offsets := make([]int, len(sentences))
	off := 0
	for i, s := range sentences {
		offsets[i] = off
		off += len(s)
	}
	notRe := regexp.MustCompile(`^(?i)\s*not\s+`)
	i := 0
	for i < len(sentences) {
		if notRe.MatchString(strings.TrimSpace(sentences[i])) {
			j := i + 1
			for j < len(sentences) && notRe.MatchString(strings.TrimSpace(sentences[j])) {
				j++
			}
			if j-i >= 2 {
				start := offsets[i]
				end := offsets[j-1] + len(sentences[j-1])
				out = append(out, types.Violation{
					RuleID:      "negation-countdown",
					StartIndex:  start,
					EndIndex:    end,
					MatchedText: text[start:end],
				})
				i = j
				continue
			}
		}
		i++
	}
	return out
}

// Function words too generic to flag as anaphora; everything else repeated
// three times at sentence-start is suspicious.
var anaphoraSingleWordSkip = map[string]bool{
	"a": true, "an": true, "the": true,
	"in": true, "on": true, "at": true, "to": true, "of": true, "for": true, "with": true, "by": true, "from": true,
	"is": true, "are": true, "was": true, "were": true,
}

var anaphoraTwoWordSkip = map[string]bool{
	"the": true, "a": true, "an": true, "it": true, "is": true,
	"in": true, "on": true, "at": true, "to": true, "of": true, "and": true, "but": true,
	"i": true, "we": true, "he": true, "she": true,
}

var anaphoraConjunctions = map[string]bool{"and": true, "but": true, "or": true}

func DetectAnaphoraAbuse(text string) []types.Violation {
	var out []types.Violation
	sentences := splitSentences(text)
	offsets := make([]int, len(sentences))
	off := 0
	for i, s := range sentences {
		offsets[i] = off
		off += len(s)
	}

	stripNonAlpha := func(w string) string {
		var b strings.Builder
		for _, r := range w {
			if unicode.IsLetter(r) {
				b.WriteRune(unicode.ToLower(r))
			}
		}
		return b.String()
	}
	normalize := func(s string) []string {
		words := strings.Fields(s)
		if len(words) > 1 && anaphoraConjunctions[stripNonAlpha(words[0])] {
			return words[1:]
		}
		return words
	}
	twoWordOpener := func(s string) string {
		words := normalize(s)
		if len(words) < 2 {
			return ""
		}
		first := stripNonAlpha(words[0])
		if anaphoraTwoWordSkip[first] || len(first) < 2 {
			return ""
		}
		return first + " " + stripNonAlpha(words[1])
	}
	singleWordOpener := func(s string) string {
		words := normalize(s)
		if len(words) < 2 {
			return ""
		}
		first := stripNonAlpha(words[0])
		if len(first) < 2 || anaphoraSingleWordSkip[first] {
			return ""
		}
		return first
	}
	flag := func(i, j int, opener string) {
		start := offsets[i]
		end := offsets[j-1] + len(sentences[j-1])
		out = append(out, types.Violation{
			RuleID:      "anaphora-abuse",
			StartIndex:  start,
			EndIndex:    end,
			MatchedText: text[start:end],
			Explanation: `"` + opener + `..." repeated ` + itoa(j-i) + " times",
		})
	}

	i := 0
	for i < len(sentences) {
		two := twoWordOpener(sentences[i])
		if two != "" {
			j := i + 1
			for j < len(sentences) && twoWordOpener(sentences[j]) == two {
				j++
			}
			if j-i >= 3 {
				flag(i, j, two)
				i = j
				continue
			}
		}
		one := singleWordOpener(sentences[i])
		if one != "" {
			j := i + 1
			for j < len(sentences) && singleWordOpener(sentences[j]) == one {
				j++
			}
			if j-i >= 3 {
				flag(i, j, one)
				i = j
				continue
			}
		}
		i++
	}
	return out
}

func DetectGerundLitany(text string) []types.Violation {
	var out []types.Violation
	sentences := splitSentences(text)
	offsets := make([]int, len(sentences))
	off := 0
	for i, s := range sentences {
		offsets[i] = off
		off += len(s)
	}
	isGerund := func(s string) bool {
		trimmed := strings.TrimSpace(s)
		words := strings.Fields(trimmed)
		return len(words) <= 8 && reCapitalGerund.MatchString(trimmed)
	}
	i := 0
	for i < len(sentences) {
		if isGerund(sentences[i]) {
			j := i + 1
			for j < len(sentences) && isGerund(sentences[j]) {
				j++
			}
			if j-i >= 2 {
				start := offsets[i]
				end := offsets[j-1] + len(sentences[j-1])
				out = append(out, types.Violation{
					RuleID:      "gerund-litany",
					StartIndex:  start,
					EndIndex:    end,
					MatchedText: text[start:end],
				})
				i = j
				continue
			}
		}
		i++
	}
	return out
}

func DetectHeresTheKicker(text string) []types.Violation {
	return detectLowerPhraseList(text, heresTheKickerPhrases, "heres-the-kicker")
}

func DetectPedagogicalAside(text string) []types.Violation {
	return detectLowerPhraseList(text, pedagogicalPhrases, "pedagogical-aside")
}

func DetectImagineWorld(text string) []types.Violation {
	return findAll(text, reImagineWorld, "imagine-world")
}

func DetectListicleTrenchCoat(text string) []types.Violation {
	var out []types.Violation
	for _, m := range reListicleTrench.FindAllStringSubmatchIndex(text, -1) {
		prefixStart, prefixEnd := m[2], m[3]
		var prefixLen int
		if prefixStart >= 0 {
			prefixLen = prefixEnd - prefixStart
		}
		out = append(out, types.Violation{
			RuleID:      "listicle-trench-coat",
			StartIndex:  m[0] + prefixLen,
			EndIndex:    m[1],
			MatchedText: text[m[0]+prefixLen : m[1]],
		})
	}
	if len(out) < 2 {
		return nil
	}
	return out
}

func DetectVagueAttribution(text string) []types.Violation {
	return detectLowerPhraseList(text, vagueAttributionPhrases, "vague-attribution")
}

func DetectBoldFirstBullets(text string) []types.Violation {
	return findAll(text, reBoldFirstBullet, "bold-first-bullets")
}

func DetectUnicodeArrows(text string) []types.Violation {
	return findAll(text, reUnicodeArrow, "unicode-arrows")
}

func DetectDespiteChallenges(text string) []types.Violation {
	return findAll(text, reDespite, "despite-challenges")
}

func DetectConceptLabel(text string) []types.Violation {
	return findAll(text, reConceptLabel, "concept-label")
}

func DetectDramaticFragment(text string) []types.Violation {
	var out []types.Violation
	paras := splitParagraphs(text)
	for _, p := range paras {
		trimmed := strings.TrimSpace(p.text)
		wordCount := len(strings.Fields(trimmed))
		if wordCount < 1 || wordCount > 4 || strings.HasSuffix(trimmed, ":") {
			continue
		}
		hasTerminal := strings.ContainsAny(trimmed, ".!?")
		isFirstPara := strings.TrimSpace(text[:p.start]) == ""
		allCapped := true
		for _, w := range strings.Fields(trimmed) {
			r, _ := firstRune(w)
			if !isDramaticLead(r) {
				allCapped = false
				break
			}
		}
		if !hasTerminal && (isFirstPara || allCapped) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:      "dramatic-fragment",
			StartIndex:  p.start,
			EndIndex:    p.start + len(p.text),
			MatchedText: trimmed,
		})
	}
	return out
}

func DetectSuperficialAnalysis(text string) []types.Violation {
	return findAll(text, reSuperficial, "superficial-analysis")
}

var reFalseRange = regexp.MustCompile(`(?i)(?:(?:doesn[\x{2019}']?t|didn[\x{2019}']?t|don[\x{2019}']?t|does\s+not|did\s+not|isn[\x{2019}']?t|wasn[\x{2019}']?t|aren[\x{2019}']?t|is\s+not|was\s+not)\s+)?(?:emerge[sd]?|comes?|came|appear[sed]*|spring[s]?|sprung|arose?|arise[s]?|materialize[sd]?|happen[sed]*|develop[sed]*|exist[sed]*)\s+from\s+nowhere`)

func DetectFalseRange(text string) []types.Violation {
	return findAll(text, reFalseRange, "false-range")
}

// ── Internal helpers ───────────────────────────────────────────────────────

func detectLowerPhraseList(text string, phrases []string, ruleID string) []types.Violation {
	lower := strings.ToLower(text)
	var out []types.Violation
	for _, phrase := range phrases {
		start := 0
		for {
			rel := strings.Index(lower[start:], phrase)
			if rel == -1 {
				break
			}
			abs := start + rel
			out = append(out, types.Violation{
				RuleID:      ruleID,
				StartIndex:  abs,
				EndIndex:    abs + len(phrase),
				MatchedText: text[abs : abs+len(phrase)],
			})
			start = abs + 1
		}
	}
	return out
}

// isDramaticLead mirrors the TS regex /^[A-Z0-9\-–—"''""\[]/ — runes that
// could head a "title-like" single-line paragraph.
func isDramaticLead(r rune) bool {
	if r == 0 {
		return false
	}
	if unicode.IsUpper(r) || unicode.IsDigit(r) {
		return true
	}
	switch r {
	case '-', '\u2013', '\u2014', '"', '\'', '\u2018', '\u2019', '\u201C', '\u201D', '[':
		return true
	}
	return false
}

func firstRune(s string) (rune, int) {
	for _, r := range s {
		return r, 1
	}
	return 0, 0
}

// itoa is a tiny stdlib-free int-to-string wrapper kept inline so detector
// files don't need strconv imports.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
