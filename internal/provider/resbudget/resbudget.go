// Package resbudget provides a shared test helper for asserting that a function stays within a per-function memory and time budget. Terraform imposes no execution timeout and no memory limit on provider-defined functions (only a large gRPC message-size ceiling), so these budgets are entirely self-imposed guardrails. A plan-time function that silently allocates gigabytes or runs for many seconds degrades every plan and apply that calls it, and an out-of-memory provider does not fail gracefully: it is OOM-killed and surfaces only as an opaque "plugin crashed".
package resbudget

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

// measureReps is how many single-call measurements to take; the minimum of each metric is reported, which removes most interference from background-goroutine allocations and scheduler jitter.
const measureReps = 3

// Check runs fn once to absorb one-time initialization, then measures the allocated bytes and wall-clock time of several subsequent single calls and reports the minimum of each. It fails tb if the allocated bytes exceed maxBytes or the time exceeds maxDur.
//
// maxBytes is measured as allocated-bytes-per-call (the runtime.MemStats.TotalAlloc delta), which is deterministic for a given Go version and CI-stable because it does not depend on CPU speed. It is a conservative proxy for peak resident memory: it counts every allocation rather than only the live ones, so the true peak is usually lower and a passing budget is a real bound.
//
// maxDur is a generous catastrophe ceiling, sized to catch pathological regressions (a function going from milliseconds to seconds) rather than to police small drift, so it does not flake on noisy shared runners.
func Check(tb testing.TB, name string, maxBytes uint64, maxDur time.Duration, fn func()) {
	tb.Helper()
	fn() // warm up: discard any one-time lazy initialization so it is not charged to the measurement

	minBytes := ^uint64(0)
	minDur := time.Duration(1<<63 - 1)
	for i := 0; i < measureReps; i++ {
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		start := time.Now()
		fn()
		elapsed := time.Since(start)
		runtime.ReadMemStats(&after)

		if b := after.TotalAlloc - before.TotalAlloc; b < minBytes {
			minBytes = b
		}
		if elapsed < minDur {
			minDur = elapsed
		}
	}

	tb.Logf("%s: %s/call, %s", name, humanBytes(minBytes), minDur.Round(time.Microsecond))
	if minBytes > maxBytes {
		tb.Errorf("%s allocated %s/call, over its budget of %s", name, humanBytes(minBytes), humanBytes(maxBytes))
	}
	if minDur > maxDur {
		tb.Errorf("%s took %s/call, over its catastrophe ceiling of %s", name, minDur.Round(time.Millisecond), maxDur)
	}
}

func humanBytes(n uint64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
