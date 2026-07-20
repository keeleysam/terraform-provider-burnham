package text

import (
	"strings"
	"testing"
	"time"

	"github.com/keeleysam/terraform-burnham/internal/provider/resbudget"
	"rsc.io/qr"
)

func TestResourceBudget(t *testing.T) {
	// levenshtein is O(len(a) * len(b)); 2048 x 2048 is a large-but-plausible input.
	a := benchString(2048)
	b := strings.ReplaceAll(a, "o", "0")
	resbudget.Check(t, "levenshtein(2048x2048)", 256<<10, time.Second, func() {
		_ = levenshteinDistance(a, b)
	})

	// qr_ascii is the closest existing "generate a visual artifact" function.
	payload := benchString(1024)
	resbudget.Check(t, "qr_ascii(1KB payload)", 2<<20, time.Second, func() {
		code, err := qr.Encode(payload, qr.M)
		if err != nil {
			t.Fatal(err)
		}
		_ = renderHalfBlock(code, 4, false)
	})
}
