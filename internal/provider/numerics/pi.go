/*
Pi digit functions (RFC 3091 §1 TCP and §2 UDP services).

  - pi_digit(n)   implements the §2.1.2 UDP reply payload format ("n:digit").
  - pi_digits(c)  implements the §1 TCP service stream (first c digits).

Both are backed by the DPD-packed 3,141,592-digit (= ⌊π × 10⁶⌋) constant in pi_data.go; no runtime computation. Out-of-range requests error explicitly per RFC's implicit "no wrong digits" stance (§5 Security Considerations).
*/

package numerics

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

// piMaxDigits is the upper bound on n for pi_digit and on count for pi_digits. RFC 3091 imposes no upper bound; this is an implementation cap matching the embedded packed digit count.
const piMaxDigits = piEmbeddedDigitCount

// piCapErrorMessage produces the RFC-aware error message we return when a caller asks for more digits than this implementation can produce. `paramName` is the function's parameter name ("n" for pi_digit, "count" for pi_digits) so the message refers to the actual argument.
func piCapErrorMessage(paramName string, received int64) string {
	return fmt.Sprintf(
		"this implementation supports %s up to %d (= floor(π × 10^6)); received %d. RFC 3091 imposes no upper bound on the digit index, but materializing more than ~π million decimal digits at plan time would meaningfully slow Terraform.",
		paramName, piMaxDigits, received,
	)
}

// ──────────────────────────────────────────────────────────────────────
// pi_digit: RFC 3091 §2.1.2 UDP reply for π
// ──────────────────────────────────────────────────────────────────────

//go:embed descriptions/pi_digit.md
var piDigitDescription string

var _ function.Function = (*PiDigitFunction)(nil)

type PiDigitFunction struct{}

func NewPiDigitFunction() function.Function { return &PiDigitFunction{} }

func (f *PiDigitFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "pi_digit"
}

func (f *PiDigitFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Return the n-th digit of π in the [RFC 3091](https://www.rfc-editor.org/rfc/rfc3091) UDP reply format",
		MarkdownDescription: piDigitDescription,
		Parameters: []function.Parameter{
			function.Int64Parameter{
				Name:        "n",
				Description: "The 1-indexed position of the desired digit following the implied leading 3.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *PiDigitFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var n int64
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &n))
	if resp.Error != nil {
		return
	}

	if n < 1 {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("RFC 3091 §2.1.1 requires n >= 1; received %d", n))
		return
	}
	if n > piMaxDigits {
		resp.Error = function.NewArgumentFuncError(0, piCapErrorMessage("n", n))
		return
	}

	reply := fmt.Sprintf("%d:%c", n, piDigitChar(n))
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &reply))
}

// ──────────────────────────────────────────────────────────────────────
// pi_digits: RFC 3091 §1 TCP service
// ──────────────────────────────────────────────────────────────────────

//go:embed descriptions/pi_digits.md
var piDigitsDescription string

var _ function.Function = (*PiDigitsFunction)(nil)

type PiDigitsFunction struct{}

func NewPiDigitsFunction() function.Function { return &PiDigitsFunction{} }

func (f *PiDigitsFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "pi_digits"
}

func (f *PiDigitsFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Return the first `count` digits of π, modeled on the [RFC 3091](https://www.rfc-editor.org/rfc/rfc3091) §1 TCP service",
		MarkdownDescription: piDigitsDescription,
		Parameters: []function.Parameter{
			function.Int64Parameter{
				Name:        "count",
				Description: "How many digits to return; count >= 0. Empty string if count = 0.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *PiDigitsFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var count int64
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &count))
	if resp.Error != nil {
		return
	}

	if count < 0 {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("count must be >= 0; received %d", count))
		return
	}
	if count > piMaxDigits {
		resp.Error = function.NewArgumentFuncError(0, piCapErrorMessage("count", count))
		return
	}

	out := piFirstNDigits(count)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}
