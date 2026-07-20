package cryptography

import (
	"testing"
	"time"

	"github.com/keeleysam/terraform-burnham/internal/provider/resbudget"
)

// The crypto keygen functions are cheap (single-digit KB, microseconds); these
// budgets exist to catch a regression that makes deterministic keygen suddenly
// expensive, not because the current cost is a concern.
func TestResourceBudget(t *testing.T) {
	resbudget.Check(t, "ecdsa_p256_key_from_seed", 64<<10, 500*time.Millisecond, func() {
		if _, err := ecdsaP256KeyFromSeed(benchSeed); err != nil {
			t.Fatal(err)
		}
	})
	resbudget.Check(t, "ed25519_key_from_seed", 64<<10, 500*time.Millisecond, func() {
		if _, err := ed25519KeyFromSeed(benchSeed[:32]); err != nil {
			t.Fatal(err)
		}
	})
}
