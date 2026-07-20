package compression

import (
	"testing"
	"time"

	"github.com/keeleysam/terraform-burnham/internal/provider/resbudget"
)

/*
Representative "commonly used" input for the compression functions: cloud-init and
user_data payloads are typically a few KB to tens of KB. These functions do not cap
input size (only iterations/quality), so the budget is asserted at this
representative size rather than a worst case. A genuinely huge input can still
exceed it, which is why the docs steer these at small payloads; if that changes,
an input-size cap should come with it.
*/
const compressionRepInput = 16 << 10 // 16 KB

func TestResourceBudget(t *testing.T) {
	in := deterministicText(compressionRepInput)

	// Zopfli is intrinsically the heaviest function burnham ships: its iterative
	// DEFLATE search allocates hundreds of MB even on small inputs. The budget has
	// headroom over the measured ~300 MB but stays well under 1 GB.
	resbudget.Check(t, "base64zopfli(16KB, 15 iterations)", 768<<20, 2*time.Second, func() {
		if _, err := zopfliGzip(in, zopfliDefaultIterations); err != nil {
			t.Fatal(err)
		}
	})
	resbudget.Check(t, "base64brotli(16KB, q11)", 128<<20, 2*time.Second, func() {
		if _, err := brotliCompress(in, brotliDefaultQuality, brotliDefaultLgwin); err != nil {
			t.Fatal(err)
		}
	})
}
