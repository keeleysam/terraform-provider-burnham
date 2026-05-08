/*
Numeric helpers that fill gaps in Terraform's built-in math: floor-modulo (`mod_floor`) and clamping (`clamp`).

`mod_floor` is the mathematical (Euclidean-flavoured) modulo operation: the result has the **sign of the divisor**, not the sign of the dividend. Terraform's built-in `%` operator follows Go's truncated-modulo convention (sign of dividend), which is the standard surprise for anyone reaching for "mod" with a negative input. RFC 5234-style "wrap into [0, b)" patterns don't work without this.

`clamp` is the obvious "bound a value into a range" function. We deliberately error when `min > max` rather than silently swapping or returning the value untouched — the call site is almost certainly buggy in that case and a hard failure is more useful than a magic recovery.
*/

package numerics

import (
	"context"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

// ──────────────────────────────────────────────────────────────────────
// mod_floor
// ──────────────────────────────────────────────────────────────────────

var _ function.Function = (*ModFloorFunction)(nil)

type ModFloorFunction struct{}

func NewModFloorFunction() function.Function { return &ModFloorFunction{} }

func (f *ModFloorFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "mod_floor"
}

func (f *ModFloorFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Floor-modulo: a − b·⌊a/b⌋, with the sign of the divisor",
		MarkdownDescription: "Returns the **floor modulo** of `a` by `b`: `a − b·⌊a/b⌋`. The result has the sign of `b`, never the sign of `a` — so for `b > 0` the result is always in `[0, b)`, exactly the \"wrap a possibly-negative index into the array length\" behaviour Python's `%` operator gives you.\n\nThis is *not* the same as Terraform's built-in `%` operator. The built-in follows Go's truncated-modulo convention, which keeps the sign of the dividend: `-7 % 3 = -1` (Terraform/Go) vs `mod_floor(-7, 3) = 2` (this function). Both are valid \"modulo\" definitions; this one is the one that makes `mod_floor(i, n)` a safe array-wrapping idiom for any integer `i`.\n\nErrors when `b == 0` (division by zero is undefined regardless of which modulo flavour you choose).",
		Parameters: []function.Parameter{
			function.NumberParameter{Name: "a", Description: "The dividend."},
			function.NumberParameter{Name: "b", Description: "The divisor; must be non-zero."},
		},
		Return: function.NumberReturn{},
	}
}

func (f *ModFloorFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var a, b *big.Float
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &a, &b))
	if resp.Error != nil {
		return
	}
	if a.IsInf() {
		resp.Error = function.NewArgumentFuncError(0, "a must be finite")
		return
	}
	if b.IsInf() {
		resp.Error = function.NewArgumentFuncError(1, "b must be finite")
		return
	}
	if b.Sign() == 0 {
		resp.Error = function.NewArgumentFuncError(1, "b must be non-zero")
		return
	}

	prec := numericPrec([]*big.Float{a, b})

	// q = a / b, truncated toward zero by Int(). For negative non-integer q we adjust to floor.
	q := new(big.Float).SetPrec(prec).Quo(a, b)
	floorInt, _ := q.Int(nil)
	floorF := new(big.Float).SetPrec(prec).SetInt(floorInt)
	if q.Sign() < 0 && floorF.Cmp(q) != 0 {
		floorInt.Sub(floorInt, big.NewInt(1))
		floorF.SetInt(floorInt)
	}

	out := new(big.Float).SetPrec(prec).Mul(b, floorF)
	out.Sub(a, out)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, out))
}

// ──────────────────────────────────────────────────────────────────────
// clamp
// ──────────────────────────────────────────────────────────────────────

var _ function.Function = (*ClampFunction)(nil)

type ClampFunction struct{}

func NewClampFunction() function.Function { return &ClampFunction{} }

func (f *ClampFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "clamp"
}

func (f *ClampFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Clamp `value` into the closed interval `[min_val, max_val]`",
		MarkdownDescription: "Returns `value` if it falls within `[min_val, max_val]`, `min_val` if `value < min_val`, and `max_val` if `value > max_val`. Equivalent to `max(min_val, min(max_val, value))` but easier to read and harder to get backwards.\n\nErrors when `min_val > max_val` — the interval is empty in that case and any return value would be a guess. Both bounds are inclusive.",
		Parameters: []function.Parameter{
			function.NumberParameter{Name: "value", Description: "The value to clamp."},
			function.NumberParameter{Name: "min_val", Description: "Lower bound (inclusive)."},
			function.NumberParameter{Name: "max_val", Description: "Upper bound (inclusive); must be >= min_val."},
		},
		Return: function.NumberReturn{},
	}
}

func (f *ClampFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value, lo, hi *big.Float
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value, &lo, &hi))
	if resp.Error != nil {
		return
	}
	if lo.Cmp(hi) > 0 {
		// `min_val` is arg 1, `max_val` is arg 2; the constraint involves both, so attribute to the upper bound (the more permissive choice when narrowing) — convention here is to flag the higher-numbered argument when both are involved.
		resp.Error = function.NewArgumentFuncError(2, fmt.Sprintf("min_val (%s) must be <= max_val (%s)", lo.Text('g', -1), hi.Text('g', -1)))
		return
	}
	prec := numericPrec([]*big.Float{value, lo, hi})
	var out *big.Float
	switch {
	case value.Cmp(lo) < 0:
		out = new(big.Float).SetPrec(prec).Copy(lo)
	case value.Cmp(hi) > 0:
		out = new(big.Float).SetPrec(prec).Copy(hi)
	default:
		out = new(big.Float).SetPrec(prec).Copy(value)
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, out))
}
