/*
Integer number-theory helpers: greatest common divisor (`gcd`) and least common multiple (`lcm`) over a list of integers.

Both operate on `list(number)` and require every element to be an integer: a gcd or lcm of a fractional value is undefined, so `gcd([1.5, 2])` is a hard error rather than a silent truncation. Empty input is an error too, consistent with the statistics functions: the gcd or lcm of no numbers is undefined.

Everything is arbitrary-precision `math/big` integer arithmetic. `gcd` folds `(*big.Int).GCD`, which since Go 1.14 accepts operands of any sign and always returns a non-negative result. `lcm` combines pairwise as `l / gcd(l, x) * |x|` so an intermediate product never overflows a fixed-width integer, and the result is built back into an exact number so a large lcm stays exact.

Conventions: `gcd` of all zeros is 0, `gcd(0, n)` is `|n|`, `lcm` is 0 whenever any element is 0, and a single-element list returns the absolute value of that element. Negatives are reduced to their absolute value, so a gcd result is always non-negative and an lcm result is always non-negative.
*/

package numerics

import (
	"context"
	_ "embed"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// parseIntegers converts each element to an exact *big.Int, rejecting any
// non-integral or infinite value with an argument error against the `numbers`
// argument (index 0). name is the function name for the message.
func parseIntegers(xs []*big.Float, name string) ([]*big.Int, *function.FuncError) {
	ints := make([]*big.Int, len(xs))
	for i, v := range xs {
		if v.IsInf() {
			return nil, function.NewArgumentFuncError(0, fmt.Sprintf("numbers[%d] is infinite; %s is defined only over integers", i, name))
		}
		iv, acc := v.Int(nil)
		if acc != big.Exact {
			return nil, function.NewArgumentFuncError(0, fmt.Sprintf("numbers[%d] (%s) is not an integer; %s is defined only over integers", i, v.Text('g', -1), name))
		}
		ints[i] = iv
	}
	return ints, nil
}

// gcdAll returns the greatest common divisor of ints. Folding from 0 gives the
// documented conventions for free: GCD(0, 0) = 0 and GCD(0, n) = |n|. GCD
// accepts operands of any sign and always returns a non-negative result.
func gcdAll(ints []*big.Int) *big.Int {
	g := new(big.Int) // starts at 0
	for _, x := range ints {
		g.GCD(nil, nil, g, x)
	}
	return g
}

// lcmAll returns the least common multiple of ints. It is 0 if any element is
// 0. Otherwise it combines pairwise as l / gcd(l, x) * |x|, dividing before
// multiplying so the intermediate value never grows larger than the final
// result. The result is non-negative.
func lcmAll(ints []*big.Int) *big.Int {
	l := big.NewInt(1)
	g := new(big.Int)
	abs := new(big.Int)
	for _, x := range ints {
		if x.Sign() == 0 {
			return big.NewInt(0)
		}
		abs.Abs(x)
		g.GCD(nil, nil, l, abs)
		l.Div(l, g)
		l.Mul(l, abs)
	}
	return l
}

// ──────────────────────────────────────────────────────────────────────
// gcd
// ──────────────────────────────────────────────────────────────────────

//go:embed descriptions/gcd.md
var gcdDescription string

var _ function.Function = (*GCDFunction)(nil)

type GCDFunction struct{}

func NewGCDFunction() function.Function { return &GCDFunction{} }

func (f *GCDFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "gcd"
}

func (f *GCDFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Greatest common divisor of a list of integers",
		MarkdownDescription: gcdDescription,
		Parameters: []function.Parameter{
			function.ListParameter{
				Name:        "numbers",
				Description: "A non-empty list of integers.",
				ElementType: types.NumberType,
			},
		},
		Return: function.NumberReturn{},
	}
}

func (f *GCDFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	xs, _, ferr := readNumberList(ctx, req)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	ints, ferr := parseIntegers(xs, "gcd")
	if ferr != nil {
		resp.Error = ferr
		return
	}
	out := new(big.Float).SetInt(gcdAll(ints))
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, out))
}

// ──────────────────────────────────────────────────────────────────────
// lcm
// ──────────────────────────────────────────────────────────────────────

//go:embed descriptions/lcm.md
var lcmDescription string

var _ function.Function = (*LCMFunction)(nil)

type LCMFunction struct{}

func NewLCMFunction() function.Function { return &LCMFunction{} }

func (f *LCMFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "lcm"
}

func (f *LCMFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Least common multiple of a list of integers",
		MarkdownDescription: lcmDescription,
		Parameters: []function.Parameter{
			function.ListParameter{
				Name:        "numbers",
				Description: "A non-empty list of integers.",
				ElementType: types.NumberType,
			},
		},
		Return: function.NumberReturn{},
	}
}

func (f *LCMFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	xs, _, ferr := readNumberList(ctx, req)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	ints, ferr := parseIntegers(xs, "lcm")
	if ferr != nil {
		resp.Error = ferr
		return
	}
	out := new(big.Float).SetInt(lcmAll(ints))
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, out))
}
