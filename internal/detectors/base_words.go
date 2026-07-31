package detectors

// irregularParticiples covers the past participles an `-ed` suffix rule cannot
// reach. Used only by DetectPassiveVoice.
var irregularParticiples = []string{
	"given", "taken", "made", "seen", "done", "written", "rewritten", "driven",
	"held", "upheld", "withheld", "built", "rebuilt", "sent", "resent", "kept",
	"left", "found", "told", "shown", "known", "thrown", "chosen", "broken",
	"drawn", "grown", "hidden", "sold", "brought", "bought", "caught", "taught",
	"understood", "overridden", "undone", "run", "read", "set", "reset", "put",
	"cut", "spread", "hit", "let", "lost", "meant", "felt", "dealt", "met",
	"paid", "said", "heard", "laid", "led", "spent", "split", "won", "begun",
	"become", "come", "gone", "eaten", "fallen", "forgotten", "frozen", "risen",
	"ridden", "spoken", "stolen", "torn", "worn", "woken", "sung", "struck",
	"stuck", "sworn", "thought", "bound", "bent", "lent", "sat", "stood",
	"dug", "hung", "shut", "quit", "cast", "sought", "fought",
}

// byNonAgentive holds the head nouns of `by` phrases that read as adverbial or
// causal rather than as an agent: "by default", "by design", "complicated by
// the fact", "passed by reference".
var byNonAgentive = map[string]bool{
	"default": true, "design": true, "hand": true, "accident": true,
	"contrast": true, "comparison": true, "chance": true, "definition": true,
	"convention": true, "necessity": true, "nature": true, "virtue": true,
	"means": true, "way": true, "example": true, "fact": true, "mistake": true,
	"force": true, "choice": true, "analogy": true, "extension": true,
	"name": true, "number": true, "reference": true, "value": true,
	"itself": true, "now": true, "then": true, "far": true, "all": true,
	"half": true, "turns": true,
}

// byDeterminers are skipped when locating the head of a `by` phrase, so
// "by the fact" resolves to "fact" rather than "the".
var byDeterminers = map[string]bool{
	"the": true, "a": true, "an": true, "this": true, "that": true,
	"its": true, "their": true, "our": true, "your": true, "his": true,
	"her": true, "any": true, "some": true, "each": true, "every": true,
}

// paddedVerbSubs is a closed list. "lets you" is deliberately absent: it is the
// recommended replacement, so flagging it would fight the fix.
var paddedVerbSubs = []plainSub{
	{"allows you to", "lets you"},
	{"allows users to", "lets users"},
	{"is able to", "can"},
	{"are able to", "can"},
	{"it is possible to", "you can"},
	{"can be used to", "can"},
	{"has the ability to", "can"},
	{"have the ability to", "can"},
	{"provides the ability to", "lets you"},
}

// hyphenCompounds are open compounds: they take a hyphen only while modifying
// a noun, and stand open in predicate position ("the tool is open source").
var hyphenCompounds = []string{
	"command line", "open source", "read only", "real time", "run time",
	"built in", "end to end", "third party", "machine readable",
	"human readable", "up to date", "out of date", "well known",
}

// hyphenNounCompounds are nouns derived from phrasal verbs. A determiner
// already makes them nouns ("the roll out starts Monday"), so unlike the open
// compounds they need no following noun to earn their hyphen. Their particles
// — "up", "off", "back", "out" — take no bare object, so nothing following
// can turn the pair back into a verb and a preposition.
var hyphenNounCompounds = []string{
	"follow up", "trade off", "fall back", "roll out", "opt in", "opt out",
}

// hyphenLocativeNouns are the phrasal-verb nouns whose second word is also a
// preposition that heads a phrase, so the first word may be a plain noun
// instead: "deposit the check in the bank", "the work around the house",
// "the drop down the shaft". Only these consult hyphenObjectStop.
var hyphenLocativeNouns = []string{
	"check in", "work around", "drop down",
}

// hyphenFollowStop lists the words that, following an open compound, show it
// is not modifying a noun: predicates ("is open source"), modals ("a third
// party may audit"), and the function words that end a noun phrase.
var hyphenFollowStop = map[string]bool{
	"is": true, "are": true, "was": true, "were": true, "be": true,
	"been": true, "being": true, "has": true, "have": true, "had": true,
	"do": true, "does": true, "did": true, "can": true, "could": true,
	"may": true, "might": true, "must": true, "shall": true, "should": true,
	"will": true, "would": true, "and": true, "or": true, "but": true,
	"so": true, "than": true, "then": true, "that": true, "which": true,
	"who": true, "whose": true, "if": true, "when": true, "while": true,
	"where": true, "because": true, "to": true, "of": true, "in": true,
	"on": true, "at": true, "by": true, "for": true, "with": true,
	"from": true, "as": true, "into": true, "onto": true, "over": true,
	"under": true, "after": true, "before": true, "during": true,
	"until": true, "unless": true, "though": true, "although": true,
	"it": true, "its": true, "this": true, "these": true, "those": true,
	"they": true, "we": true, "you": true, "he": true, "she": true,
	"there": true, "here": true, "not": true, "no": true, "also": true,
	"still": true, "just": true, "only": true, "always": true,
	"never": true, "often": true, "now": true, "yet": true, "again": true,
	"too": true, "very": true, "per": true, "via": true, "plus": true,
	"goes": true, "comes": true, "works": true, "runs": true, "starts": true,
	"stops": true, "looks": true, "seems": true, "means": true, "needs": true,
	"takes": true, "gets": true, "makes": true, "lets": true, "gives": true,
	"shows": true, "says": true, "uses": true, "exists": true, "applies": true,
	"becomes": true, "remains": true, "returns": true, "belongs": true,
	"stays": true, "wins": true, "counts": true, "matters": true,
}

// hyphenObjectStop lists what may follow a phrasal-verb noun to show its
// second word is a preposition taking an object rather than half a compound:
// "deposit the check in the bank" against "complete the check in before noon".
var hyphenObjectStop = map[string]bool{
	"the": true, "a": true, "an": true, "this": true, "that": true,
	"these": true, "those": true, "my": true, "your": true, "his": true,
	"her": true, "its": true, "our": true, "their": true,
	"it": true, "them": true, "him": true, "me": true, "us": true,
	"you": true,
}

// expletiveNegations carve out "There are no guarantees." — an existential
// negation states something, where "There is a thing that…" only delays it.
var expletiveNegations = map[string]bool{
	"no": true, "not": true, "nothing": true, "never": true,
	"little": true, "few": true, "neither": true, "nor": true,
}

// nominalizationSubs is a closed set of light-verb frames. Suffix-based
// detection (-tion/-ment/-ance) is deliberately absent: it measured 23-30
// hits per 1000 words at >=24% false positives on "documentation",
// "implementation", "configuration", "argument", and "environment".
// The "consideration" family belongs to falseConclusionPhrases.
var nominalizationSubs = []plainSub{
	{"make a decision", "decide"},
	{"makes a decision", "decides"},
	{"made a decision", "decided"},
	{"perform an analysis", "analyze"},
	{"provide clarification", "clarify"},
	{"reach a conclusion", "conclude"},
	{"come to a conclusion", "conclude"},
	{"has an impact on", "affects"},
	{"have an impact on", "affect"},
	{"is an indication of", "indicates"},
	{"place a restriction on", "limits"},
	{"make use of", "use"},
	{"makes use of", "uses"},
}
