/*
Approximate-pi (22/7) digit functions (RFC 3091 §1.1 TCP and §2.2 UDP "approximate" services).

  - pi_approximate_digit(n)  → "n:digit" per RFC §2.1.2 (UDP reply for 22/7)
  - pi_approximate_digits(c) → first c digits of 22/7 (TCP stream for 22/7)

22/7 = 3.142857142857… is a period-6 repeating decimal: long division of 22 by 7 yields the cycle "142857" starting after the decimal point. So the n-th digit is "142857"[(n-1) mod 6] — constant-time lookup, no big math required for the lookup itself, and no upper bound on n.

The single-digit function uses NumberParameter (arbitrary precision) so truly enormous n (up to ~10^150 in Terraform's 512-bit number type) is supported. The bulk function uses Int64Parameter for `count` because the returned string must be materialized in memory; we don't try to handle counts larger than int can address.
*/

package numerics

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

// approximateCycle is the 6-digit repeat of 22/7 = 3.142857142857….
const approximateCycle = "142857"

// approximateDigitChar returns the n-th decimal digit of 22/7 *following* the decimal point. n must be >= 1; out-of-range is the caller's job.
//
// Computes (n - 1) mod 6 via big.Int.Mod so n can be arbitrarily large.
func approximateDigitChar(n *big.Int) byte {
	nm1 := new(big.Int).Sub(n, big.NewInt(1))
	mod := new(big.Int).Mod(nm1, big.NewInt(int64(len(approximateCycle))))
	return approximateCycle[mod.Int64()]
}

// approximateFirstNDigits returns the first n digits of 22/7 *following* the decimal point. n must be in [0, math.MaxInt].
func approximateFirstNDigits(n int64) string {
	if n <= 0 {
		return ""
	}
	var b strings.Builder
	b.Grow(int(n))
	for i := int64(0); i < n; i++ {
		b.WriteByte(approximateCycle[i%int64(len(approximateCycle))])
	}
	return b.String()
}

// ──────────────────────────────────────────────────────────────────────
// pi_approximate_digit — RFC 3091 §2.2 UDP reply for 22/7
// ──────────────────────────────────────────────────────────────────────

var _ function.Function = (*PiApproximateDigitFunction)(nil)

type PiApproximateDigitFunction struct{}

func NewPiApproximateDigitFunction() function.Function { return &PiApproximateDigitFunction{} }

func (f *PiApproximateDigitFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "pi_approximate_digit"
}

func (f *PiApproximateDigitFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Return the n-th digit of 22/7 in the [RFC 3091](https://www.rfc-editor.org/rfc/rfc3091) UDP reply format",
		MarkdownDescription: "Returns the n-th decimal digit of 22/7 *following* the decimal point, formatted as the [RFC 3091 §2.1.2](https://www.rfc-editor.org/rfc/rfc3091#section-2.1.2) UDP reply payload `reply = nth_digit \":\" DIGIT`. This is the \"approximate service\" of RFC 3091 §1.1/§2.2 — long division of 22 by 7 gives `3.142857142857…`, a period-6 repeating cycle of `\"142857\"`.\n\nExamples:\n- `pi_approximate_digit(1)` → `\"1:1\"`\n- `pi_approximate_digit(7)` → `\"7:1\"` (cycle wraps to start of `\"142857\"`)\n- `pi_approximate_digit(100)` → `\"100:8\"`\n\n**No upper bound on n.** Because 22/7 cycles with period 6, the n-th digit is just `\"142857\"[(n-1) mod 6]` — a constant-time lookup. n can be arbitrarily large (up to ~10^150 in Terraform's 512-bit number type) and the function returns instantly.",
		Parameters: []function.Parameter{
			function.NumberParameter{
				Name:        "n",
				Description: "The 1-indexed position of the desired digit following the implied leading 3.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *PiApproximateDigitFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var n *big.Float
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &n))
	if resp.Error != nil {
		return
	}

	// Defensive: (*big.Float).Int returns nil for ±Inf, which would panic on the Sign()/String() calls below. Terraform's number parser shouldn't produce Inf, but check explicitly so the contract holds even if the framework changes.
	if n.IsInf() {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("n must be finite; received %s", n.Text('g', -1)))
		return
	}
	nInt, accuracy := n.Int(nil)
	if accuracy != big.Exact {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("n must be a whole number; received %s", n.Text('g', -1)))
		return
	}
	if nInt.Sign() < 1 {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("RFC 3091 §2.1.1 requires n >= 1; received %s", nInt.String()))
		return
	}

	reply := fmt.Sprintf("%s:%c", nInt.String(), approximateDigitChar(nInt))
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &reply))
}

// ──────────────────────────────────────────────────────────────────────
// pi_approximate_digits — RFC 3091 §1.1 TCP service for 22/7
// ──────────────────────────────────────────────────────────────────────

var _ function.Function = (*PiApproximateDigitsFunction)(nil)

type PiApproximateDigitsFunction struct{}

func NewPiApproximateDigitsFunction() function.Function { return &PiApproximateDigitsFunction{} }

func (f *PiApproximateDigitsFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "pi_approximate_digits"
}

func (f *PiApproximateDigitsFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Return the first `count` digits of 22/7, modeled on the [RFC 3091](https://www.rfc-editor.org/rfc/rfc3091) §1.1 TCP approximate service",
		MarkdownDescription: "Returns the first `count` decimal digits of 22/7 *following* the decimal point as a single ASCII string. Models the [RFC 3091 §1.1](https://www.rfc-editor.org/rfc/rfc3091#section-1.1) TCP approximate service, which streams `\"starting with the most significant digit following the decimal point\"` — no seek operation, so this function takes only `count`.\n\nExample:\n- `pi_approximate_digits(12)` → `\"142857142857\"` (the 6-digit cycle, twice)\n\nBecause 22/7 is a period-6 repeating decimal, output for any count `c` is just `\"142857\"` repeated and truncated. Count is bounded only by what Go's `int` and your machine's memory can hold.",
		Parameters: []function.Parameter{
			function.Int64Parameter{
				Name:        "count",
				Description: "How many digits to return; count >= 0. Empty string if count = 0.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *PiApproximateDigitsFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var count int64
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &count))
	if resp.Error != nil {
		return
	}

	if count < 0 {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("count must be >= 0; received %d", count))
		return
	}

	out := approximateFirstNDigits(count)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}
