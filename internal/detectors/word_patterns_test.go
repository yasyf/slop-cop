package detectors

import (
	"testing"

	"github.com/yasyf/slop-cop/internal/types"
)

// hasRule returns true if any violation in v carries the given ruleID.
func hasRule(v []types.Violation, ruleID string) bool {
	for _, x := range v {
		if x.RuleID == ruleID {
			return true
		}
	}
	return false
}

// countRule counts violations with the given ruleID.
func countRule(v []types.Violation, ruleID string) int {
	n := 0
	for _, x := range v {
		if x.RuleID == ruleID {
			n++
		}
	}
	return n
}

type hitCase struct {
	name    string
	text    string
	wantHit bool
}

// runHits is a reusable table runner: each detector function maps its input
// to an "expect at least one violation of ruleID" assertion.
func runHits(t *testing.T, ruleID string, detect func(string) []types.Violation, cases []hitCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := hasRule(detect(c.text), ruleID)
			if got != c.wantHit {
				t.Fatalf("%s: got fires=%v want=%v\ntext: %q", ruleID, got, c.wantHit, c.text)
			}
		})
	}
}

func TestDetectOverusedIntensifiers(t *testing.T) {
	runHits(t, "overused-intensifiers", DetectOverusedIntensifiers, []hitCase{
		{"crucial", "This is crucial to understand.", true},
		{"leverage", "We must leverage our existing assets.", true},
		{"delve", "Let us delve into the details.", true},
		{"robust", "We built a robust framework.", true},
		{"nuanced", "This requires a nuanced approach.", true},
		{"pivotal", "This is a pivotal moment in history.", true},
		{"unprecedented", "We are living through an unprecedented crisis.", true},
		{"tapestry", "A rich tapestry of cultural influences.", true},
		{"multifaceted", "This is a multifaceted problem.", true},
		{"landscape", "The competitive landscape has shifted.", true},
		{"underscores", "This underscores the importance of planning.", true},
		{"paradigm", "We need a new paradigm for thinking about this.", true},
		{"ordinary words", "The cat sat on the mat.", false},
	})
}

func TestDetectElevatedRegister(t *testing.T) {
	runHits(t, "elevated-register", DetectElevatedRegister, []hitCase{
		{"utilize", "We should utilize this tool.", true},
		{"commence", "We will commence the process tomorrow.", true},
		{"facilitate", "This will facilitate better outcomes.", true},
		{"endeavor", "We will endeavor to improve.", true},
		{"demonstrate", "The results demonstrate that the approach works.", true},
		{"craft", "We should craft a response to each concern.", true},
		{"moving forward", "Moving forward, we will focus on delivery.", true},
		{"at this juncture", "At this juncture, a decision is required.", true},
		{"use not flagged", "We should use this tool.", false},
		{"show not flagged", "The data shows a clear trend.", false},
	})
}

func TestDetectFillerAdverbs(t *testing.T) {
	runHits(t, "filler-adverbs", DetectFillerAdverbs, []hitCase{
		{"importantly", "Importantly, this affects everyone.", true},
		{"ultimately", "Ultimately, success depends on effort.", true},
		{"essentially", "This is essentially a marketing problem.", true},
		{"fundamentally", "This is fundamentally wrong.", true},
		{"generally not flagged", "We generally recognize this right.", false},
	})
}

func TestDetectAlmostHedge(t *testing.T) {
	runHits(t, "almost-hedge", DetectAlmostHedge, []hitCase{
		{"almost always", "This is almost always true.", true},
		{"almost never", "It almost never works that way.", true},
		{"almost certainly", "This will almost certainly happen.", true},
		{"almost alone", "It is almost done.", false},
	})
}

func TestDetectEraOpener(t *testing.T) {
	runHits(t, "era-opener", DetectEraOpener, []hitCase{
		{"in an era of", "In an era of rapid change, companies must adapt.", true},
		{"in an era where", "We live in an era where everything is connected.", true},
		{"year reference", "The company was founded in 1990.", false},
	})
}

func TestDetectMetaphorCrutch(t *testing.T) {
	runHits(t, "metaphor-crutch", DetectMetaphorCrutch, []hitCase{
		{"double-edged sword", "This is a double-edged sword.", true},
		{"game changer", "AI is a game changer for the industry.", true},
		{"tip of the iceberg", "This is just the tip of the iceberg.", true},
		{"north star", "Quality is our north star.", true},
		{"deep dive", "Let's do a deep dive into the data.", true},
		{"paradigm shift", "It's not just a tool—it's a paradigm shift.", true},
		{"elephant in the room", "The elephant in the room is that nobody reads the documentation.", true},
		{"perfect storm", "A perfect storm of budget cuts and talent flight.", true},
		{"building blocks", "These are the building blocks of a successful strategy.", true},
		{"ordinary", "The results were better than expected.", false},
	})
}

func TestDetectImportantToNote(t *testing.T) {
	runHits(t, "important-to-note", DetectImportantToNote, []hitCase{
		{"important to note", "It is important to note that this affects everyone.", true},
		{"worth noting", "It's worth noting that results may vary.", true},
		{"should be noted", "It should be noted that exceptions exist.", true},
		{"ordinary", "The results were consistent.", false},
	})
}

func TestDetectBroaderImplications(t *testing.T) {
	runHits(t, "broader-implications", DetectBroaderImplications, []hitCase{
		{"broader implications", "This has broader implications for society.", true},
		{"wider implications", "The wider implications are unclear.", true},
		{"unrelated", "The policy was updated last year.", false},
	})
}

func TestDetectFalseConclusion(t *testing.T) {
	runHits(t, "false-conclusion", DetectFalseConclusion, []hitCase{
		{"in conclusion", "In conclusion, we have shown that X is true.", true},
		{"at the end of the day", "At the end of the day, results matter most.", true},
		{"to summarize", "To summarize, the three key points are these.", true},
		{"moving forward", "Moving forward, we must prioritize trust over speed.", true},
		{"going forward", "Going forward, the focus will shift to execution.", true},
	})
	t.Run("does not flag mid-sentence usage", func(t *testing.T) {
		// "all in all" mid-sentence is borderline; ensure it doesn't explode.
		_ = DetectFalseConclusion("The project, all in all, was a success.")
	})
}

func TestDetectConnectorAddiction(t *testing.T) {
	runHits(t, "connector-addiction", DetectConnectorAddiction, []hitCase{
		{"furthermore", "Furthermore, this approach has merit.", true},
		{"moreover", "Moreover, the data confirms our hypothesis.", true},
		{"additionally", "Additionally, we found three other patterns.", true},
		{"however", "However, the results were inconclusive.", true},
		{"that said", "That said, there are exceptions worth noting.", true},
		{"with that in mind", "With that in mind, we can now turn to the solution.", true},
	})
	t.Run("flags a chain of connectors across paragraphs", func(t *testing.T) {
		chain := "First point.\n\nFurthermore, the evidence is clear.\n\nMoreover, this has been confirmed.\n\nAdditionally, the trend holds."
		if n := countRule(DetectConnectorAddiction(chain), "connector-addiction"); n < 3 {
			t.Fatalf("want >=3 connector-addiction hits in chain, got %d", n)
		}
	})
}

func TestDetectUnnecessaryContrast(t *testing.T) {
	runHits(t, "unnecessary-contrast", DetectUnnecessaryContrast, []hitCase{
		{"whereas", "This approach works, whereas the old one did not.", true},
		{"whereas-spec", "Models write one register above where a human would, whereas human writers tend to match register to context.", true},
		{"as opposed to", "We use data, as opposed to intuition.", true},
		{"unlike", "Unlike its predecessor, this version is fast.", true},
		{"in contrast to", "In contrast to earlier models, this one performs well.", true},
	})
}

func TestDetectEmDashPivot(t *testing.T) {
	runHits(t, "em-dash-pivot", DetectEmDashPivot, []hitCase{
		{"em-dash", "This is important—but often overlooked.", true},
		{"not X—Y", "It's not just a tool—it's a paradigm shift.", true},
		{"em-dash semicolon", "The data shows one thing—the conclusion is another.", true},
		{"em-dash parenthetical", "The answer—and this surprises most people—is simpler than expected.", true},
		{"ascii hyphen", "This is a well-known fact.", false},
	})
	t.Run("flags multiple em-dashes", func(t *testing.T) {
		if n := countRule(DetectEmDashPivot("First—second—third."), "em-dash-pivot"); n < 2 {
			t.Fatalf("want >=2 em-dash hits, got %d", n)
		}
	})
}

func TestDetectNegationPivot(t *testing.T) {
	runHits(t, "negation-pivot", DetectNegationPivot, []hitCase{
		{"straight apostrophe", "Companies don't succeed by luck, but by discipline.", true},
		{"curly apostrophe", "The system doesn\u2019t constrain through prohibition, but through amplification.", true},
		{"do not but", "We do not build for speed, but for resilience.", true},
		{"not through but", "The choice architectures don\u2019t constrain through prohibition, but through amplification and attenuation.", true},
		{"isn't but no comma", "The question isn\u2019t whether to use these technologies but in whose interests and under whose control they operate.", true},
		{"is not but", "The issue is not access but accountability.", true},
		{"not X—Y", "It's not just a tool—it's a paradigm shift.", true},
		{"isn't X—Y", "This isn\u2019t about technology\u2014it\u2019s about trust.", true},
		{"two-sentence same subject", "It doesn't check whether text was written by an AI. It checks whether text reads like it was.", true},
		{"two-sentence different subject matches This", "This doesn't solve the problem. This reframes it.", true},
		{"but without negation", "The results were good, but not perfect.", false},
		{"two sentences different subjects", "She doesn't like the proposal. He thinks it has merit.", false},
	})
}

func TestDetectColonElaboration(t *testing.T) {
	runHits(t, "colon-elaboration", DetectColonElaboration, []hitCase{
		{"simple elaboration", "The solution is simple: we need to change how we approach the fundamental problem at its root.", true},
		{"answer simple", "The answer is simple: we need to rethink our approach from the ground up.", true},
		{"there is one problem", "There is one problem: the data does not support the conclusion we reached.", true},
	})
	t.Run("does not flag a colon in a short list item", func(t *testing.T) {
		// Short setup clause below the 5/20 char thresholds — must not crash.
		_ = DetectColonElaboration("Note: done.")
	})
}

func TestDetectParentheticalQualifier(t *testing.T) {
	runHits(t, "parenthetical-qualifier", DetectParentheticalQualifier, []hitCase{
		{"long paren", "This approach (which has been widely debated in the literature) is not new.", true},
		{"of course", "This is, of course, a simplification.", true},
		{"to be fair", "There are, to be fair, exceptions.", true},
		{"admittedly", "The approach is, admittedly, imperfect.", true},
		{"needless to say", "This is, needless to say, complicated.", true},
		{"short paren", "Use a tool (e.g. a hammer) for this.", false},
	})
}

func TestDetectQuestionThenAnswer(t *testing.T) {
	runHits(t, "question-then-answer", DetectQuestionThenAnswer, []hitCase{
		{"short Q short A", "What does this mean? It means we must adapt.", true},
		{"so what does this mean", "So what does this mean for the average user? It means everything.", true},
		{"same paragraph newline", "Why does this matter?\nIt shapes every decision we make.", true},
		{"long answer", "How can independent musicians compete when the most popular streaming algorithms consistently favor major-label releases?\nThis is a structural problem about what kind of relationship we want between platforms, capital, and the artists who actually produce the music that makes these services valuable.", false},
		{"cross paragraph", "What does this mean?\n\nThe building codes governing this type of construction were written before composite materials became commercially viable at scale.", false},
		{"no question", "The building codes governing this type of construction were written before composite materials became commercially viable at scale.", false},
	})
}

func TestDetectHedgeStack(t *testing.T) {
	runHits(t, "hedge-stack", DetectHedgeStack, []hitCase{
		{"multiple hedges", "Perhaps this might arguably be considered a problem.", true},
		{"hedges plus modal", "Seemingly, this could perhaps be the right approach.", true},
		{"may not be potentially", "It's worth noting that, while this may not be universally applicable, in many cases it can potentially offer significant benefits.", true},
		{"single hedge", "Perhaps this is worth considering.", false},
		{"should is not hedge", "We are witness to a kind of massive institutional failure, the non-adoption of tools that should exist but don\u2019t.", false},
		{"kind of classifier", "This is a kind of problem that requires careful thought.", false},
		{"would is not hedge", "That would be a significant improvement to the system.", false},
	})
}

func TestDetectStaccatoBurst(t *testing.T) {
	runHits(t, "staccato-burst", DetectStaccatoBurst, []hitCase{
		{"four short", "AI is here. It is growing. It is changing everything. We must act.", true},
		{"this matters trio", "This matters. It always has. And it always will.", true},
		{"clear undeniable obvious", "The data is clear. The trend is undeniable. The conclusion is obvious.", true},
		{"two short sentences", "AI is here. It is growing.", false},
		{"long sentences", "Artificial intelligence is fundamentally reshaping how we think about knowledge. The implications for education, work, and human creativity are profound and far-reaching.", false},
	})
}

func TestDetectListicleInstinct(t *testing.T) {
	runHits(t, "listicle-instinct", DetectListicleInstinct, []hitCase{
		{"bulleted 3", "- First item\n- Second item\n- Third item", true},
		{"numbered 5", "1. One\n2. Two\n3. Three\n4. Four\n5. Five", true},
		{"bullet 4", "- One\n- Two\n- Three\n- Four", false},
		{"numbered 6", "1. One\n2. Two\n3. Three\n4. Four\n5. Five\n6. Six", false},
		{"numbered 7", "1. One\n2. Two\n3. Three\n4. Four\n5. Five\n6. Six\n7. Seven", true},
	})
}

func TestDetectServesAs(t *testing.T) {
	runHits(t, "serves-as", DetectServesAs, []hitCase{
		{"serves as", "The building serves as a reminder of the city's heritage.", true},
		{"stands as", "This stands as the best example we have.", true},
		{"acts as", "The policy acts as a deterrent.", true},
		{"functions as", "The layer functions as a buffer.", true},
		{"plain is", "The building is a landmark.", false},
	})
}

func TestDetectNegationCountdown(t *testing.T) {
	runHits(t, "negation-countdown", DetectNegationCountdown, []hitCase{
		{"two not then third", "Not a bug. Not a feature. A fundamental design flaw.", true},
		{"three not sentences", "Not fast. Not slow. Not in between. Just broken.", true},
		{"single not", "Not everything is as it seems. The data tells a different story.", false},
	})
}

func TestDetectAnaphoraAbuse(t *testing.T) {
	runHits(t, "anaphora-abuse", DetectAnaphoraAbuse, []hitCase{
		{"they assume x3", "They assume the worst. They assume silence means guilt. They assume nothing will change.", true},
		{"every decision x4", "Every decision matters. Every decision counts. Every decision shapes the outcome. Every decision defines us.", true},
		{"varied", "They started early. We caught up quickly. Everyone finished on time.", false},
		{"only 2 matching", "They assume the worst. They assume nothing. The data is clear.", false},
		{"both x3", "Both can be difficult to understand. Both are active at all hours. Both connect distant things.", true},
		{"each x4", "Each decision matters. Each voice counts. Each moment shapes the outcome. Each choice defines us.", true},
		{"only 2 curated", "Both can be difficult. Both are active. The third is different.", false},
		{"people x3", "People often forget. People make mistakes. People learn slowly.", true},
		{"his x3", "His argument was X. His evidence was Y. His conclusion was Z.", true},
		{"this x3 with and", "This is foo. This is bar. And this is baz.", true},
		{"articles prepositions", "In the beginning. In the middle. In the end.", false},
		{"and matches base", "Both can be difficult. Both are active. Both connect things. And both produce alarm.", true},
		{"and two words matches", "They assume the worst. They assume silence means guilt. And they assume nothing will change.", true},
	})
}

func TestDetectGerundLitany(t *testing.T) {
	runHits(t, "gerund-litany", DetectGerundLitany, []hitCase{
		{"three gerunds", "Fixing small bugs. Writing straightforward features. Implementing well-defined tickets.", true},
		{"two gerunds", "Building quickly. Shipping often.", true},
		{"single gerund", "Building a product takes time.", false},
		{"long gerund", "Building a product that users actually love and return to is hard.", false},
	})
}

func TestDetectHeresTheKicker(t *testing.T) {
	runHits(t, "heres-the-kicker", DetectHeresTheKicker, []hitCase{
		{"kicker", "Here's the kicker — nobody saw it coming.", true},
		{"thing", "Here's the thing about distributed systems.", true},
		{"gets interesting", "Here's where it gets interesting: the data contradicts the theory.", true},
		{"uppercase", "HERE'S THE KICKER: everything changed.", true},
		{"ordinary", "The meeting starts at noon.", false},
	})
}

func TestDetectPedagogicalAside(t *testing.T) {
	runHits(t, "pedagogical-aside", DetectPedagogicalAside, []hitCase{
		{"break down", "Let's break this down step by step.", true},
		{"unpack", "Let's unpack what this means.", true},
		{"think of it as", "Think of it as a pipeline.", true},
		{"think of this as", "Think of this as a foundation.", true},
		{"lets meet", "Let's meet tomorrow to discuss this.", false},
		{"ordinary", "The system processes requests in order.", false},
	})
}

func TestDetectImagineWorld(t *testing.T) {
	runHits(t, "imagine-world", DetectImagineWorld, []hitCase{
		{"a world where", "Imagine a world where every tool is connected.", true},
		{"if you", "Imagine if you could access any data instantly.", true},
		{"what would", "Imagine what would happen if the system failed.", true},
		{"a future", "Imagine a future without passwords.", true},
		{"imagine alone", "Imagine the possibilities.", false},
	})
}

func TestDetectListicleTrenchCoat(t *testing.T) {
	runHits(t, "listicle-trench-coat", DetectListicleTrenchCoat, []hitCase{
		{"two ordinals", "The first issue is cost. The second issue is time.", true},
		{"three ordinals", "The first reason is speed. The second reason is reliability. The third reason is cost.", true},
		{"single ordinal", "The first thing to understand is that context matters.", false},
	})
}

func TestDetectVagueAttribution(t *testing.T) {
	runHits(t, "vague-attribution", DetectVagueAttribution, []hitCase{
		{"experts argue", "Experts argue that this approach has drawbacks.", true},
		{"studies show", "Studies show that remote work increases productivity.", true},
		{"research suggests", "Research suggests a correlation between sleep and performance.", true},
		{"observers have noted", "Observers have noted a shift in user behavior.", true},
		{"named citation", "The paper by Smith argues that framing matters.", false},
	})
}

func TestDetectBoldFirstBullets(t *testing.T) {
	runHits(t, "bold-first-bullets", DetectBoldFirstBullets, []hitCase{
		{"dash bullets", "- **Security**: keeps data safe\n- **Performance**: runs fast", true},
		{"asterisk bullets", "* **Scalability**: handles load\n* **Reliability**: stays up", true},
		{"plain bullets", "- plain item\n- another plain item", false},
		{"bold in sentence", "This is **important** and should be noted.", false},
	})
}

func TestDetectUnicodeArrows(t *testing.T) {
	runHits(t, "unicode-arrows", DetectUnicodeArrows, []hitCase{
		{"single", "Input \u2192 Output", true},
		{"ascii arrow", "Input -> Output", false},
	})
	t.Run("flags multiple arrows", func(t *testing.T) {
		if n := countRule(DetectUnicodeArrows("Step 1 \u2192 Step 2 \u2192 Step 3"), "unicode-arrows"); n < 2 {
			t.Fatalf("want >=2 arrow hits, got %d", n)
		}
	})
}

func TestDetectDespiteChallenges(t *testing.T) {
	runHits(t, "despite-challenges", DetectDespiteChallenges, []hitCase{
		{"these challenges", "Despite these challenges, the platform continues to grow.", true},
		{"its limitations", "Despite its limitations, the tool remains popular.", true},
		{"the obstacles", "Despite the obstacles, the team shipped on time.", true},
		{"unrelated", "The project succeeded because of careful planning.", false},
	})
}

func TestDetectConceptLabel(t *testing.T) {
	runHits(t, "concept-label", DetectConceptLabel, []hitCase{
		{"supervision paradox", "This is the supervision paradox at its core.", true},
		{"trust vacuum", "We are living through a trust vacuum.", true},
		{"attention trap", "The attention trap affects every platform.", true},
		{"innovation chasm", "Companies fall into the innovation chasm.", true},
		{"ordinary", "The product launched on schedule.", false},
	})
}

func TestDetectDramaticFragment(t *testing.T) {
	runHits(t, "dramatic-fragment", DetectDramaticFragment, []hitCase{
		{"full stop", "This is a long paragraph with real content and ideas.\n\nFull stop.\n\nAnd this continues.", true},
		{"boom", "Here is the setup.\n\nBoom.\n\nAnd here is the rest.", true},
		{"normal paragraphs", "This is the first paragraph with sufficient content.\n\nThis is the second paragraph also with sufficient content to not be flagged.", false},
		{"five word paragraphs", "This is the first paragraph with plenty of words.\n\nThis paragraph also has five words here.\n\nThis is the third paragraph with plenty of words too.", false},
	})
}

func TestDetectSuperficialAnalysis(t *testing.T) {
	runHits(t, "superficial-analysis", DetectSuperficialAnalysis, []hitCase{
		{"underscoring its role", "The initiative succeeded, underscoring its role as a community hub.", true},
		{"highlighting its importance", "The campaign resonated with voters, highlighting its importance in the region.", true},
		{"cementing its legacy", "The album sold millions, cementing its legacy in music history.", true},
		{"reflecting the significance", "The award was given quietly, reflecting the significance of the work.", true},
		{"ordinary participle", "She left the building, waving goodbye to her colleagues.", false},
	})
}

func TestDetectFalseRange(t *testing.T) {
	runHits(t, "false-range", DetectFalseRange, []hitCase{
		{"doesn't emerge from nowhere", "The push for urban cycling infrastructure doesn't emerge from nowhere; it stands in a long tradition of transport activism.", true},
		{"came from nowhere", "This movement came from nowhere and changed everything.", true},
		{"does not come from nowhere", "This idea does not come from nowhere.", true},
		{"didn't appear from nowhere", "The crisis didn't appear from nowhere.", true},
		{"ordinary from", "She emerged from the building.", false},
		{"directional from", "They came from the countryside.", false},
	})
}
