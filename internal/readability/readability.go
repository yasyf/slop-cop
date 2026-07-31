// Package readability scores prose with the Flesch reading-ease and
// Flesch-Kincaid grade-level formulas.
//
// Every number it returns is an estimate, which is what [Report.Estimated]
// records. The syllable counter is a vowel-group heuristic with a few
// English-specific corrections rather than a pronunciation dictionary, so it
// undercounts hiatus ("idea" reads as two syllables, not three). No claim is
// made that these scores match any other implementation's.
package readability

import (
	"math"
	"regexp"
	"strings"
	"unicode"
)

// MinWords is the shortest prose [Analyze] will score. Below it one long word
// moves the grade by a whole level, so the number would be noise.
const MinWords = 100

// Report is an advisory readability summary. It is never a violation.
type Report struct {
	FleschReadingEase  float64 `json:"flesch_reading_ease"`
	FleschKincaidGrade float64 `json:"flesch_kincaid_grade"`
	Words              int     `json:"words"`
	Sentences          int     `json:"sentences"`
	Syllables          int     `json:"syllables"`
	Estimated          bool    `json:"estimated"`
}

// Analyze scores text, returning nil for prose shorter than [MinWords].
func Analyze(text string) *Report {
	words := strings.Fields(text)
	if len(words) < MinWords {
		return nil
	}
	syllableCount := 0
	for _, w := range words {
		syllableCount += syllables(w)
	}
	sentenceCount := countSentences(text)
	ease, grade := scores(len(words), sentenceCount, syllableCount)
	return &Report{
		FleschReadingEase:  ease,
		FleschKincaidGrade: grade,
		Words:              len(words),
		Sentences:          sentenceCount,
		Syllables:          syllableCount,
		Estimated:          true,
	}
}

// scores applies both formulas, rounded to two decimals. Any input with at
// least MinWords words carries at least one sentence, so neither ratio can
// divide by zero.
func scores(words, sentences, syllables int) (ease, grade float64) {
	wordsPerSentence := float64(words) / float64(sentences)
	syllablesPerWord := float64(syllables) / float64(words)
	ease = 206.835 - 1.015*wordsPerSentence - 84.6*syllablesPerWord
	grade = 0.39*wordsPerSentence + 11.8*syllablesPerWord - 15.59
	return round2(ease), round2(grade)
}

func round2(f float64) float64 { return math.Round(f*100) / 100 }

var (
	blankLine   = regexp.MustCompile(`\n\s*\n`)
	sentenceEnd = regexp.MustCompile(`[.!?]+\s+`)
	abbrevTail  = regexp.MustCompile(`(?:e\.g\.|i\.e\.|Fig\.|\b[A-Za-z]\.)$`)
)

// countSentences segments text the way internal/detectors segments it before
// applying the long-sentence rule: paragraphs on blank lines, sentences on
// terminal punctuation followed by whitespace, abbreviations re-merged. Those
// helpers are unexported and sit inside the awnist provenance boundary
// (NOTICE), so the rules are restated here instead of shared;
// TestSegmentationMatchesLongSentence pins the two implementations together.
func countSentences(text string) int {
	n := 0
	for _, p := range blankLine.Split(text, -1) {
		if strings.TrimSpace(p) == "" {
			continue
		}
		n += countInParagraph(p)
	}
	return n
}

// countInParagraph counts boundaries without materializing sentences. The
// abbreviation test reads only the chunk that just closed, never the merged
// group it joins — a group's tail is its last chunk's tail, and re-testing the
// whole group makes a paragraph of N merging chunks O(N²) to scan.
func countInParagraph(p string) int {
	n, end := 0, 0
	merging := false
	for _, idx := range sentenceEnd.FindAllStringIndex(p, -1) {
		chunk := p[end:idx[1]]
		end = idx[1]
		if strings.TrimSpace(chunk) == "" {
			continue
		}
		if !merging {
			n++
		}
		merging = abbrevTail.MatchString(strings.TrimRight(chunk, " \t\r\n"))
	}
	if !merging && end < len(p) && strings.TrimSpace(p[end:]) != "" {
		n++
	}
	return n
}

// syllables estimates the syllable count of one word: vowel groups, less a
// silent trailing "e", plus corrections for a syllabic "-le" ("sim-ple"), a
// vowel before "-ing" ("be-ing"), and a syllabic final "m" ("rhy-thm").
// A word carrying no letters counts none; any other word counts at least one.
func syllables(word string) int {
	letters := make([]rune, 0, len(word))
	for _, r := range strings.ToLower(word) {
		if unicode.IsLetter(r) {
			letters = append(letters, r)
		}
	}
	if len(letters) == 0 {
		return 0
	}

	n := 0
	prev := false
	for i := range letters {
		cur := isVowel(letters, i)
		if cur && !prev {
			n++
		}
		prev = cur
	}

	w := string(letters)
	last := len(letters) - 1
	syllabicLE := strings.HasSuffix(w, "le") && last >= 2 && !isVowel(letters, last-2)
	if strings.HasSuffix(w, "e") && n > 1 && !syllabicLE {
		n--
	}
	if strings.HasSuffix(w, "ing") && last >= 3 && isVowel(letters, last-3) {
		n++
	}
	if strings.HasSuffix(w, "sm") || strings.HasSuffix(w, "thm") {
		n++
	}
	if n < 1 {
		return 1
	}
	return n
}

// isVowel reports whether letters[i] is a vowel, counting "y" everywhere but
// word-initially ("rhythm" has one, "yes" has one).
func isVowel(letters []rune, i int) bool {
	switch letters[i] {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	case 'y':
		return i > 0
	}
	return false
}
