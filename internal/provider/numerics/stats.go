/*
Statistics functions for lists of numbers: mean, median, percentile, variance, stddev, mode.

All operate on `list(number)`, accept Terraform's arbitrary-precision number type, and return a number (or for `mode`, a sorted list of numbers, since the data may be multimodal). Empty input is always an error: a statistic of zero observations is undefined.

Variance and standard deviation use the **population** formulas: divide the sum of squared deviations by N, not N-1. This matches numpy's default (`ddof=0`). Callers who want sample statistics (Bessel's correction) can multiply variance by N/(N-1) explicitly. Mixing those defaults silently is the kind of foot-gun the rest of this provider tries to avoid.

Percentile uses the linear-interpolation method (Type 7 in Hyndman & Fan, the default in numpy, R, and Excel's PERCENTILE.INC): index = p/100 × (N - 1), interpolate between the two nearest observations when p does not land on an integer index.

All arithmetic is `*big.Float` at a precision derived from the inputs (see `numericPrec`) so a list of 64-bit doubles or a list of arbitrary-precision integers both yield exact-as-possible answers without arbitrary truncation.
*/

package numerics

import (
	"context"
	_ "embed"
	"fmt"
	"math/big"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// numericPrec returns the working precision (in bits) for big.Float arithmetic over the given inputs. We pick the maximum of (a) each input's own precision, (b) a 256-bit floor that comfortably exceeds IEEE 754 double, (c) some headroom proportional to N so summing many values does not lose digits, capped at (d) 4096 bits (≈1233 decimal digits) so a 10K-element list does not blow into multi-megabyte arithmetic for what is realistically a 30-digit answer. Shared with math.go's clamp / mod_floor: the same precision policy applies wherever the numerics family does mixed-input big.Float arithmetic.
func numericPrec(xs []*big.Float) uint {
	const floor = 256
	const ceiling = 4096
	max := uint(floor)
	for _, x := range xs {
		if p := x.Prec(); p > max {
			max = p
		}
	}
	// Use 64-bit math then narrow so the addition cannot wrap on absurdly long lists; numericPrec is bounded by `ceiling` immediately afterward so the narrowing is safe in practice.
	if extra := uint64(64) + 8*uint64(len(xs)); extra > uint64(max) {
		if extra > ceiling {
			extra = ceiling
		}
		max = uint(extra)
	}
	if max > ceiling {
		max = ceiling
	}
	return max
}

// validateNumberList rejects empty lists and infinite elements. Null elements are rejected by the framework before Run sees them (ListParameter does not set AllowNullValue), so we don't re-check that here.
func validateNumberList(xs []*big.Float) *function.FuncError {
	if len(xs) == 0 {
		return function.NewArgumentFuncError(0, "numbers must contain at least one value; received an empty list")
	}
	for i, v := range xs {
		if v.IsInf() {
			return function.NewArgumentFuncError(0, fmt.Sprintf("numbers[%d] is infinite; statistics over infinity are undefined", i))
		}
	}
	return nil
}

// readNumberList pulls a single positional list-of-number argument out of a function request, validates non-empty + finite, and returns the slice plus the working precision.
func readNumberList(ctx context.Context, req function.RunRequest) ([]*big.Float, uint, *function.FuncError) {
	var raw []*big.Float
	if err := req.Arguments.Get(ctx, &raw); err != nil {
		return nil, 0, err
	}
	if ferr := validateNumberList(raw); ferr != nil {
		return nil, 0, ferr
	}
	return raw, numericPrec(raw), nil
}

// sumWithPrec returns Σ xs at precision prec.
func sumWithPrec(xs []*big.Float, prec uint) *big.Float {
	out := new(big.Float).SetPrec(prec)
	for _, x := range xs {
		out.Add(out, x)
	}
	return out
}

// meanWithPrec returns Σ xs / |xs| at precision prec. Caller must ensure |xs| > 0.
func meanWithPrec(xs []*big.Float, prec uint) *big.Float {
	out := sumWithPrec(xs, prec)
	n := new(big.Float).SetPrec(prec).SetInt64(int64(len(xs)))
	return out.Quo(out, n)
}

// populationVariance returns Σ (x - μ)² / N at precision prec.
func populationVariance(xs []*big.Float, prec uint) *big.Float {
	mean := meanWithPrec(xs, prec)
	acc := new(big.Float).SetPrec(prec)
	tmp := new(big.Float).SetPrec(prec)
	for _, x := range xs {
		tmp.Sub(x, mean)
		tmp.Mul(tmp, tmp)
		acc.Add(acc, tmp)
	}
	n := new(big.Float).SetPrec(prec).SetInt64(int64(len(xs)))
	return acc.Quo(acc, n)
}

// sortedCopy returns a new slice containing xs in ascending order. Original is not modified.
func sortedCopy(xs []*big.Float) []*big.Float {
	out := make([]*big.Float, len(xs))
	copy(out, xs)
	sort.Slice(out, func(i, j int) bool { return out[i].Cmp(out[j]) < 0 })
	return out
}

// ──────────────────────────────────────────────────────────────────────
// mean
// ──────────────────────────────────────────────────────────────────────

//go:embed descriptions/mean.md
var meanDescription string

var _ function.Function = (*MeanFunction)(nil)

type MeanFunction struct{}

func NewMeanFunction() function.Function { return &MeanFunction{} }

func (f *MeanFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "mean"
}

func (f *MeanFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Arithmetic mean (average) of a list of numbers",
		MarkdownDescription: meanDescription,
		Parameters: []function.Parameter{
			function.ListParameter{
				Name:        "numbers",
				Description: "A non-empty list of numbers.",
				ElementType: types.NumberType,
			},
		},
		Return: function.NumberReturn{},
	}
}

func (f *MeanFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	xs, prec, ferr := readNumberList(ctx, req)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	out := meanWithPrec(xs, prec)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, out))
}

// ──────────────────────────────────────────────────────────────────────
// median
// ──────────────────────────────────────────────────────────────────────

//go:embed descriptions/median.md
var medianDescription string

var _ function.Function = (*MedianFunction)(nil)

type MedianFunction struct{}

func NewMedianFunction() function.Function { return &MedianFunction{} }

func (f *MedianFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "median"
}

func (f *MedianFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Median (50th percentile) of a list of numbers",
		MarkdownDescription: medianDescription,
		Parameters: []function.Parameter{
			function.ListParameter{
				Name:        "numbers",
				Description: "A non-empty list of numbers.",
				ElementType: types.NumberType,
			},
		},
		Return: function.NumberReturn{},
	}
}

func (f *MedianFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	xs, prec, ferr := readNumberList(ctx, req)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	sorted := sortedCopy(xs)
	n := len(sorted)
	var out *big.Float
	if n%2 == 1 {
		out = new(big.Float).SetPrec(prec).Copy(sorted[n/2])
	} else {
		out = new(big.Float).SetPrec(prec).Add(sorted[n/2-1], sorted[n/2])
		out.Quo(out, new(big.Float).SetPrec(prec).SetInt64(2))
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, out))
}

// ──────────────────────────────────────────────────────────────────────
// percentile
// ──────────────────────────────────────────────────────────────────────

//go:embed descriptions/percentile.md
var percentileDescription string

var _ function.Function = (*PercentileFunction)(nil)

type PercentileFunction struct{}

func NewPercentileFunction() function.Function { return &PercentileFunction{} }

func (f *PercentileFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "percentile"
}

func (f *PercentileFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Percentile of a list of numbers, by linear interpolation",
		MarkdownDescription: percentileDescription,
		Parameters: []function.Parameter{
			function.ListParameter{
				Name:        "numbers",
				Description: "A non-empty list of numbers.",
				ElementType: types.NumberType,
			},
			function.NumberParameter{
				Name:        "p",
				Description: "The percentile to compute, in [0, 100].",
			},
		},
		Return: function.NumberReturn{},
	}
}

func (f *PercentileFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var xs []*big.Float
	var p *big.Float
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &xs, &p))
	if resp.Error != nil {
		return
	}
	if ferr := validateNumberList(xs); ferr != nil {
		resp.Error = ferr
		return
	}
	if p.IsInf() {
		resp.Error = function.NewArgumentFuncError(1, "p must be a finite number in [0, 100]")
		return
	}
	zero := new(big.Float).SetInt64(0)
	hundred := new(big.Float).SetInt64(100)
	if p.Cmp(zero) < 0 || p.Cmp(hundred) > 0 {
		resp.Error = function.NewArgumentFuncError(1, fmt.Sprintf("p must be in [0, 100]; received %s", p.Text('g', -1)))
		return
	}

	prec := numericPrec(xs)
	if pp := p.Prec(); pp > prec {
		prec = pp
	}
	sorted := sortedCopy(xs)
	n := len(sorted)
	if n == 1 {
		out := new(big.Float).SetPrec(prec).Copy(sorted[0])
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, out))
		return
	}

	/*
		h = (p / 100) * (N - 1). Compute it as an exact rational, not a binary big.Float, so the "does the index land exactly on an observation" decision is exact.
		Example: percentile([0..25], 4) has h = 0.04 * 25 = 1 exactly, but 0.04 is not representable in binary, so a big.Float h comes out as 0.999...868. That would miss the integer-index fast path and interpolate, returning 0.999... instead of the exact observation x[1] = 1.
		p is finite (checked above), so p.Rat is exact; N-1 and 100 are integers, so hRat is the exact value of h.
	*/
	pRat := new(big.Rat)
	p.Rat(pRat)
	hRat := new(big.Rat).Mul(pRat, big.NewRat(int64(n-1), 100))

	// hRat is non-negative (we validated p ≥ 0), so integer division of numerator by denominator is the floor.
	floorInt := new(big.Int).Quo(hRat.Num(), hRat.Denom())
	idx := floorInt.Int64()

	// Exact integer index, no interpolation needed. Covers p = 0, p = 100, and any p that lands on an observation.
	if hRat.IsInt() {
		out := new(big.Float).SetPrec(prec).Copy(sorted[idx])
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, out))
		return
	}

	// frac = h - ⌊h⌋, taken exactly then rounded to the working precision for interpolation.
	fracRat := new(big.Rat).Sub(hRat, new(big.Rat).SetInt(floorInt))
	frac := new(big.Float).SetPrec(prec).SetRat(fracRat)

	lo := sorted[idx]
	hi := sorted[idx+1]
	out := new(big.Float).SetPrec(prec).Sub(hi, lo)
	out.Mul(out, frac)
	out.Add(out, lo)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, out))
}

// ──────────────────────────────────────────────────────────────────────
// variance (population)
// ──────────────────────────────────────────────────────────────────────

//go:embed descriptions/variance.md
var varianceDescription string

var _ function.Function = (*VarianceFunction)(nil)

type VarianceFunction struct{}

func NewVarianceFunction() function.Function { return &VarianceFunction{} }

func (f *VarianceFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "variance"
}

func (f *VarianceFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Population variance of a list of numbers",
		MarkdownDescription: varianceDescription,
		Parameters: []function.Parameter{
			function.ListParameter{
				Name:        "numbers",
				Description: "A non-empty list of numbers.",
				ElementType: types.NumberType,
			},
		},
		Return: function.NumberReturn{},
	}
}

func (f *VarianceFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	xs, prec, ferr := readNumberList(ctx, req)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	out := populationVariance(xs, prec)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, out))
}

// ──────────────────────────────────────────────────────────────────────
// stddev (population)
// ──────────────────────────────────────────────────────────────────────

//go:embed descriptions/stddev.md
var stddevDescription string

var _ function.Function = (*StddevFunction)(nil)

type StddevFunction struct{}

func NewStddevFunction() function.Function { return &StddevFunction{} }

func (f *StddevFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "stddev"
}

func (f *StddevFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Population standard deviation of a list of numbers",
		MarkdownDescription: stddevDescription,
		Parameters: []function.Parameter{
			function.ListParameter{
				Name:        "numbers",
				Description: "A non-empty list of numbers.",
				ElementType: types.NumberType,
			},
		},
		Return: function.NumberReturn{},
	}
}

func (f *StddevFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	xs, prec, ferr := readNumberList(ctx, req)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	v := populationVariance(xs, prec)
	out := new(big.Float).SetPrec(prec).Sqrt(v)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, out))
}

// ──────────────────────────────────────────────────────────────────────
// mode
// ──────────────────────────────────────────────────────────────────────

//go:embed descriptions/mode.md
var modeDescription string

var _ function.Function = (*ModeFunction)(nil)

type ModeFunction struct{}

func NewModeFunction() function.Function { return &ModeFunction{} }

func (f *ModeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "mode"
}

func (f *ModeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Most-frequent value(s) in a list of numbers",
		MarkdownDescription: modeDescription,
		Parameters: []function.Parameter{
			function.ListParameter{
				Name:        "numbers",
				Description: "A non-empty list of numbers.",
				ElementType: types.NumberType,
			},
		},
		Return: function.ListReturn{ElementType: types.NumberType},
	}
}

func (f *ModeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	xs, _, ferr := readNumberList(ctx, req)
	if ferr != nil {
		resp.Error = ferr
		return
	}

	sorted := sortedCopy(xs)
	type bucket struct {
		value *big.Float
		count int
	}
	var buckets []bucket
	for _, v := range sorted {
		if len(buckets) > 0 && buckets[len(buckets)-1].value.Cmp(v) == 0 {
			buckets[len(buckets)-1].count++
		} else {
			buckets = append(buckets, bucket{value: v, count: 1})
		}
	}
	maxCount := 0
	for _, b := range buckets {
		if b.count > maxCount {
			maxCount = b.count
		}
	}
	var modes []*big.Float
	// All-unique data with more than one element has no mode: returning every input would mislead callers using `mode` to detect repetition. Single-element input is the degenerate case where the value is trivially "the mode".
	if maxCount == 1 && len(buckets) > 1 {
		modes = []*big.Float{}
	} else {
		for _, b := range buckets {
			if b.count == maxCount {
				modes = append(modes, b.value)
			}
		}
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, modes))
}
