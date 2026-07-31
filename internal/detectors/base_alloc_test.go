//go:build !race

package detectors

import (
	"runtime"
	"testing"
)

// Merging must not copy the accumulated group per chunk. At 92KB the copying
// version allocated ~184MB and took 6.9s; joining once per group allocates
// under 2MB. The bound is deliberately loose — it separates linear from
// quadratic, not one linear implementation from another.
//
// Excluded from -race builds: the race detector's shadow-memory bookkeeping
// inflates TotalAlloc by roughly two orders of magnitude, which drowns the
// signal this bound exists to read. CI runs the un-raced suite too, so the
// guard still gates every push.
func TestMergeAbbrevDoesNotCopyPerChunk(t *testing.T) {
	doc := abbrevDoc(4000)
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	DetectLongSentence(doc)
	runtime.ReadMemStats(&after)
	allocated := after.TotalAlloc - before.TotalAlloc
	if limit := uint64(50 * len(doc)); allocated > limit {
		t.Fatalf("DetectLongSentence allocated %d bytes over a %d byte document, want under %d",
			allocated, len(doc), limit)
	}
}
