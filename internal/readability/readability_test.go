package readability

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/yasyf/slop-cop/internal/detectors"
)

func TestScoresKnownValues(t *testing.T) {
	cases := []struct {
		name                        string
		words, sentences, syllables int
		wantEase, wantGrade         float64
	}{
		// 20 words/sentence, 1.5 syllables/word:
		//   206.835 - 1.015*20 - 84.6*1.5 = 59.635
		//   0.39*20 + 11.8*1.5 - 15.59    = 9.91
		{"twenty word sentences", 100, 5, 150, 59.635, 9.91},
		// 10 words/sentence, 1.2 syllables/word:
		//   206.835 - 10.15 - 101.52 = 95.165
		//   3.9 + 14.16 - 15.59      = 2.47
		{"short and simple", 200, 20, 240, 95.165, 2.47},
		// 25 words/sentence, 2 syllables/word:
		//   206.835 - 25.375 - 169.2 = 12.26
		//   9.75 + 23.6 - 15.59      = 17.76
		{"long and latinate", 100, 4, 200, 12.26, 17.76},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ease, grade := scores(c.words, c.sentences, c.syllables)
			if math.Abs(ease-c.wantEase) > 0.01 {
				t.Errorf("ease = %v, want %v", ease, c.wantEase)
			}
			if math.Abs(grade-c.wantGrade) > 0.01 {
				t.Errorf("grade = %v, want %v", grade, c.wantGrade)
			}
		})
	}
}

func TestSyllables(t *testing.T) {
	cases := map[string]int{
		"a": 1, "I": 1, "the": 1, "code": 1, "queue": 1, "style": 1, "whole": 1,
		"thing": 1, "bring": 1, "strength": 1,
		"simple": 2, "little": 2, "able": 2, "people": 2, "being": 2, "doing": 2,
		"seeing": 2, "using": 2, "rhythm": 2, "prism": 2, "sentence": 2,
		"during": 2, "running": 2, "hyphen": 2,
		"syllable": 3, "criticism": 4, "algorithm": 4, "readability": 5,
		"detector": 3, "paragraph": 3, "chocolate": 3,
		// no letters at all
		"—": 0, "|": 0, "": 0,
		// no vowels: the floor applies
		"hmm": 1, "tsk": 1,
	}
	for word, want := range cases {
		t.Run(word, func(t *testing.T) {
			if got := syllables(word); got != want {
				t.Errorf("syllables(%q) = %d, want %d", word, got, want)
			}
		})
	}
}

// The counter is a heuristic, and these are the words it is known to miss.
// The test records the estimate rather than the dictionary answer so a change
// in behaviour is visible.
func TestSyllablesKnownMisses(t *testing.T) {
	cases := []struct {
		word            string
		estimate, truth int
	}{
		{"idea", 2, 3},
		{"every", 3, 2},
		{"business", 3, 2},
	}
	for _, c := range cases {
		t.Run(c.word, func(t *testing.T) {
			if got := syllables(c.word); got != c.estimate {
				t.Errorf("syllables(%q) = %d, want the known estimate %d (dictionary says %d)",
					c.word, got, c.estimate, c.truth)
			}
		})
	}
}

func TestCountSentences(t *testing.T) {
	cases := []struct {
		name string
		text string
		want int
	}{
		{"one sentence", "The tool reads a file.", 1},
		{"three sentences", "One runs. Two runs. Three runs.", 3},
		{"no terminal punctuation", "The tool reads a file", 1},
		{"across paragraphs", "One runs.\n\nTwo runs.\n\nThree runs.", 3},
		{"heading and body", "# Title\n\nThe tool reads a file. Then it stops.", 3},
		{"e.g. does not split", "Run it on a file, e.g. README.md, and stop.", 1},
		{"i.e. does not split", "Use the base layer, i.e. the clarity rules, first.", 1},
		{"initials do not split", "The author is J. R. R. Tolkien.", 1},
		{"abbreviation plus a real break", "Run it, e.g. here, and stop. Then check.", 2},
		{"ellipsis is one break", "It stopped... Then it resumed.", 2},
		{"empty", "", 0},
		{"whitespace only", "   \n\n  \t ", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := countSentences(c.text); got != c.want {
				t.Errorf("countSentences(%q) = %d, want %d", c.text, got, c.want)
			}
		})
	}
}

// The sentence segmentation here must agree with the long-sentence detector's,
// or the score and the violations disagree about the same document. Every
// sentence in the fixture runs past 40 words, so the detector emits exactly one
// violation per sentence it sees.
func TestSegmentationMatchesLongSentence(t *testing.T) {
	long := func(lead string) string {
		return lead + " " + strings.TrimSpace(strings.Repeat("and then the analyzer hands the bytes to the next detector ", 4)) + "."
	}
	fixtures := map[string]string{
		"three sentences":      long("The pipeline reads the file") + " " + long("The report is written") + " " + long("The runner exits"),
		"with an abbreviation": long("The pipeline reads a file, e.g. README.md, and stops") + " " + long("The runner exits"),
		"across paragraphs":    long("The pipeline reads the file") + "\n\n" + long("The runner exits"),
	}
	for name, doc := range fixtures {
		t.Run(name, func(t *testing.T) {
			want := len(detectors.DetectLongSentence(doc))
			if want == 0 {
				t.Fatal("the fixture tripped no long-sentence violation; it is not exercising anything")
			}
			if got := countSentences(doc); got != want {
				t.Errorf("countSentences = %d, long-sentence saw %d sentences", got, want)
			}
		})
	}
}

const simplePassage = `The tool reads a file. It splits the text into words. It counts the words.
It counts the sentences. It counts the syllables. It then does the math. The score
comes back as two numbers. One is a grade. One is an ease score. Both are guesses.
The code is short. The rules are plain. A short word is easy. A long word is hard.
A short line is easy. A long line is hard. That is the whole idea. The rest is math.
You can read the code. You can change the code. You can test the code. It is all here.
The tool does not care what you write. It just counts. Then it tells you what it found.`

const densePassage = `The instrumentation subsystem consolidates heterogeneous telemetry
acquisitions into a normalized representation, whereupon subsequent analytical
transformations may be applied without additional reconciliation of the underlying
observational semantics, notwithstanding the considerable architectural divergence
exhibited by the contributing infrastructural components. Consequently, the
aforementioned normalization procedure necessarily incorporates configurable
compatibility accommodations, each parameterized independently, so that
organizations possessing idiosyncratic instrumentation topologies retain the
capability of participating in the consolidated analytical environment without
undertaking a comprehensive reimplementation of their existing observability
apparatus, which experience demonstrates to be prohibitively expensive. The
resulting consolidated representation subsequently facilitates comparative
analysis across organizational boundaries, since the normalization procedure
guarantees that equivalent observations acquire equivalent representations
irrespective of the instrumentation particulars responsible for generating them,
an invariant the implementation maintains through continuous validation against
representative acquisition samples collected from participating organizations.`

func TestAnalyzeShortInputReturnsNil(t *testing.T) {
	if got := Analyze(strings.Repeat("word ", MinWords-1)); got != nil {
		t.Fatalf("Analyze under MinWords returned %+v, want nil", got)
	}
	got := Analyze(strings.Repeat("word ", MinWords))
	if got == nil {
		t.Fatal("Analyze at MinWords returned nil")
	}
	if got.Words != MinWords {
		t.Errorf("Words = %d, want %d", got.Words, MinWords)
	}
	if !got.Estimated {
		t.Error("Estimated is false; every score this package produces is an estimate")
	}
}

func TestAnalyzeSimplePassage(t *testing.T) {
	got := Analyze(simplePassage)
	if got == nil {
		t.Fatal("Analyze returned nil on a passage over MinWords")
	}
	if want := len(strings.Fields(simplePassage)); got.Words != want {
		t.Errorf("Words = %d, want %d", got.Words, want)
	}
	if got.Sentences != 25 {
		t.Errorf("Sentences = %d, want 25", got.Sentences)
	}
	ease, grade := scores(got.Words, got.Sentences, got.Syllables)
	if got.FleschReadingEase != ease || got.FleschKincaidGrade != grade {
		t.Errorf("report carries (%v, %v), formula gives (%v, %v)",
			got.FleschReadingEase, got.FleschKincaidGrade, ease, grade)
	}
	// Flesch-Kincaid is unbounded below — five-word sentences of one-syllable
	// words score under zero by construction — so only the ceiling is asserted.
	if got.FleschKincaidGrade > 4 {
		t.Errorf("grade = %v for one-clause sentences of short words, want a low one", got.FleschKincaidGrade)
	}
	if got.FleschReadingEase < 85 {
		t.Errorf("ease = %v for one-clause sentences of short words, want a high one", got.FleschReadingEase)
	}
}

func TestDenseProseScoresHarder(t *testing.T) {
	simple, dense := Analyze(simplePassage), Analyze(densePassage)
	if simple == nil || dense == nil {
		t.Fatal("both passages must clear MinWords")
	}
	if dense.FleschKincaidGrade <= simple.FleschKincaidGrade {
		t.Errorf("dense grade %v is not above simple grade %v",
			dense.FleschKincaidGrade, simple.FleschKincaidGrade)
	}
	if dense.FleschReadingEase >= simple.FleschReadingEase {
		t.Errorf("dense ease %v is not below simple ease %v",
			dense.FleschReadingEase, simple.FleschReadingEase)
	}
}

func TestAbbreviationDoesNotInflateSentences(t *testing.T) {
	plain := simplePassage + " The tool reads the config and stops."
	abbrev := simplePassage + " The tool reads the config, e.g. slop.json, and stops."
	a, b := Analyze(plain), Analyze(abbrev)
	if a == nil || b == nil {
		t.Fatal("both passages must clear MinWords")
	}
	if a.Sentences != b.Sentences {
		t.Errorf("abbreviation changed the sentence count: %d without, %d with", a.Sentences, b.Sentences)
	}
}

func TestDegenerateInput(t *testing.T) {
	cases := map[string]string{
		"empty":                   "",
		"whitespace":              " \t\n\n   \n ",
		"one word":                "word",
		"no terminal punctuation": strings.Repeat("word ", 200),
		"no vowels":               strings.Repeat("hmm ", 200),
		"punctuation only":        strings.Repeat("— | * ", 200),
		"one long token":          strings.Repeat("a", 5000),
	}
	for name, text := range cases {
		t.Run(name, func(t *testing.T) {
			got := Analyze(text)
			if len(strings.Fields(text)) < MinWords {
				if got != nil {
					t.Fatalf("got %+v, want nil below MinWords", got)
				}
				return
			}
			if got == nil {
				t.Fatal("got nil at or above MinWords")
			}
			if got.Sentences < 1 {
				t.Errorf("Sentences = %d, want at least 1", got.Sentences)
			}
			if math.IsNaN(got.FleschReadingEase) || math.IsInf(got.FleschReadingEase, 0) {
				t.Errorf("ease = %v", got.FleschReadingEase)
			}
			if math.IsNaN(got.FleschKincaidGrade) || math.IsInf(got.FleschKincaidGrade, 0) {
				t.Errorf("grade = %v", got.FleschKincaidGrade)
			}
		})
	}
}

// The abbreviation test must read only the chunk that just closed. Re-testing
// the whole merged group made a 92KB document of merging sentences take 6.8s;
// it now takes single-digit milliseconds. The bound separates linear from
// quadratic with room for a loaded machine.
func TestCountSentencesStaysLinear(t *testing.T) {
	doc := strings.Repeat("Run it on a file, e.g. ", 4000)
	start := time.Now()
	countSentences(doc)
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("countSentences took %v on %d bytes, want well under 2s", elapsed, len(doc))
	}
}

func BenchmarkCountSentencesAbbrev(b *testing.B) {
	doc := strings.Repeat("Run it on a file, e.g. ", 4000)
	b.SetBytes(int64(len(doc)))
	for i := 0; i < b.N; i++ {
		countSentences(doc)
	}
}
