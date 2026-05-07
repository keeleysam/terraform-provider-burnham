package numerics

import (
	"testing"

	"github.com/keeleysam/terraform-burnham/internal/provider/numerics/internal/chudnovsky"
)

// crossValidatePrefix is the number of leading digits that must match between the embedded packed table and a freshly-computed Chudnovsky run during normal `go test ./...`.
//
// This is a small prefix on purpose: a fast (~3 ms) sanity check to catch DPD-decode or off-by-one bugs in piFirstNDigits without paying the ~25-second cost of regenerating all ⌊π × 10⁶⌋ = 3,141,592 digits inside the test process. The exhaustive whole-table cross-check happens in CI via `go generate ./...` followed by `git diff -- pi_packed.bin` — see .github/workflows/test.yml. Together the two guarantees cover the same ground without doubling the work.
const crossValidatePrefix = 10_000

// TestPackedAgainstChudnovsky cross-validates the first crossValidatePrefix embedded-table digits against an independent Chudnovsky computation.
//
// Note this does NOT verify the entire table — that's a CI responsibility.
// What this catches:
//   - the runtime DPD decoder produces correct digits,
//   - the packed file isn't accidentally truncated or corrupted in a way that affects the prefix.
//
// What only the CI guard catches:
//   - someone edits cmd/genpi or chudnovsky and forgets to commit the regenerated pi_packed.bin (drift in the tail of the table).
func TestPackedAgainstChudnovsky(t *testing.T) {
	want := chudnovsky.PiDigits(crossValidatePrefix)
	got := piFirstNDigits(crossValidatePrefix)
	if got != want {
		// Find first divergence to make the failure useful.
		for i := 0; i < len(got) && i < len(want); i++ {
			if got[i] != want[i] {
				t.Fatalf("packed disagrees with Chudnovsky at index %d: packed=%q chudnovsky=%q", i, got[i], want[i])
			}
		}
		t.Fatalf("packed/Chudnovsky lengths differ: packed=%d, chudnovsky=%d", len(got), len(want))
	}
}
