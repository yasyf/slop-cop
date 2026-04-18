// Package detectors hosts the client-side (instant) detection pipeline ported
// from src/detectors of awnist/slop-cop. Each exported Detect* function runs
// a single rule; [RunClient] runs them all and deduplicates the result.
package detectors

import (
	"strconv"

	"github.com/yasyf/slop-cop/internal/types"
)

// RunClient executes every client-side detector over text and returns a
// merged, deduplicated slice of violations (equivalent to the TS
// `runClientDetectors`).
func RunClient(text string) []types.Violation {
	detectors := []func(string) []types.Violation{
		DetectOverusedIntensifiers,
		DetectElevatedRegister,
		DetectFillerAdverbs,
		DetectAlmostHedge,
		DetectEraOpener,
		DetectMetaphorCrutch,
		DetectImportantToNote,
		DetectBroaderImplications,
		DetectFalseConclusion,
		DetectConnectorAddiction,
		DetectUnnecessaryContrast,
		DetectEmDashPivot,
		DetectNegationPivot,
		DetectColonElaboration,
		DetectParentheticalQualifier,
		DetectQuestionThenAnswer,
		DetectHedgeStack,
		DetectStaccatoBurst,
		DetectListicleInstinct,
		DetectServesAs,
		DetectNegationCountdown,
		DetectAnaphoraAbuse,
		DetectGerundLitany,
		DetectHeresTheKicker,
		DetectPedagogicalAside,
		DetectImagineWorld,
		DetectListicleTrenchCoat,
		DetectVagueAttribution,
		DetectBoldFirstBullets,
		DetectUnicodeArrows,
		DetectDespiteChallenges,
		DetectConceptLabel,
		DetectDramaticFragment,
		DetectSuperficialAnalysis,
		DetectFalseRange,
	}
	var all []types.Violation
	for _, d := range detectors {
		all = append(all, d(text)...)
	}
	return Deduplicate(all)
}

// Deduplicate removes exact duplicates by (ruleId, start, end); overlapping
// spans from different rules are kept.
func Deduplicate(in []types.Violation) []types.Violation {
	seen := make(map[string]struct{}, len(in))
	out := in[:0]
	for _, v := range in {
		key := v.RuleID + ":" + strconv.Itoa(v.StartIndex) + ":" + strconv.Itoa(v.EndIndex)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, v)
	}
	// Copy to release the backing array capacity we aliased from `in`.
	result := make([]types.Violation, len(out))
	copy(result, out)
	return result
}
