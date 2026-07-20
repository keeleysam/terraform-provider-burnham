package numerics

import (
	"testing"
	"time"

	"github.com/keeleysam/terraform-burnham/internal/provider/resbudget"
)

// pi_digits and pi_approximate_digits cap their count at piEmbeddedDigitCount, so
// the worst case a caller can request is bounded and directly testable. Both are
// O(n) over that cap: a few MB and single-digit ms.
func TestResourceBudget(t *testing.T) {
	resbudget.Check(t, "pi_digits(max)", 16<<20, time.Second, func() {
		_ = piFirstNDigits(piEmbeddedDigitCount)
	})
	resbudget.Check(t, "pi_approximate_digits(max)", 16<<20, time.Second, func() {
		_ = approximateFirstNDigits(piApproximateMaxDigits)
	})
}
