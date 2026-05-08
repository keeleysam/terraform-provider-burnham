/*
Statistics functions for lists of numbers: mean, median, percentile, variance, stddev, mode.

All operate on `list(number)`, accept Terraform's arbitrary-precision number type, and return a number (or for `mode`, a sorted list of numbers — the data may be multimodal). Empty input is always an error: a statistic of zero observations is undefined.

Variance and standard deviation use the **population** formulas — divide the sum of squared deviations by N, not N-1. This matches numpy's default (`ddof=0`). Callers who want sample statistics (Bessel's correction) can multiply variance by N/(N-1) explicitly. Mixing those defaults silently is the kind of foot-gun the rest of this provider tries to avoid.

Percentile uses the linear-interpolation method (Type 7 in Hyndman & Fan, the default in numpy, R, and Excel's PERCENTILE.INC): index = p/100 × (N - 1), interpolate between the two nearest observations when p does not land on an integer index.

All arithmetic is `*big.Float` at a precision derived from the inputs (see `statsPrec`) so a list of 64-bit doubles or a list of arbitrary-precision integers both yield exact-as-possible answers without arbitrary truncation.
*/

package numerics

import (
	"context"
	"fmt"
	"math/big"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// statsPrec returns the working precision (in bits) for big.Float arithmetic over the given inputs. We pick the maximum of (a) each input's own precision, (b) a 256-bit floor that comfortably exceeds IEEE 754 double, (c) some headroom proportional to N so summing many values does not lose digits, capped at (d) 4096 bits (≈1233 decimal digits) so a 10K-element list does not blow into multi-megabyte arithmetic for what is realistically a 30-digit answer.
func statsPrec(xs []*big.Float) uint {
	const floor = 256
	const cap = 4096
	max := uint(floor)
	for _, x := range xs {
		if p := x.Prec(); p > max {
			max = p
		}
	}
	if extra := uint(64 + 8*len(xs)); extra > max {
		max = extra
	}
	if max > cap {
		max = cap
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
	return raw, statsPrec(raw), nil
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

var _ function.Function = (*MeanFunction)(nil)

type MeanFunction struct{}

func NewMeanFunction() function.Function { return &MeanFunction{} }

func (f *MeanFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "mean"
}

func (f *MeanFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Arithmetic mean (average) of a list of numbers",
		MarkdownDescription: "Returns Σ x / N, the arithmetic mean of `numbers`. Errors when `numbers` is empty.\n\nUse this when you want a plain average. For weighted means, geometric means, or trimmed means, do the weighting explicitly in HCL — the goal of this function is the unambiguous canonical definition.",
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

var _ function.Function = (*MedianFunction)(nil)

type MedianFunction struct{}

func NewMedianFunction() function.Function { return &MedianFunction{} }

func (f *MedianFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "median"
}

func (f *MedianFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Median (50th percentile) of a list of numbers",
		MarkdownDescription: "Returns the median of `numbers`. For odd N this is the middle value of the sorted list; for even N it is the arithmetic mean of the two central values. Equivalent to `percentile(numbers, 50)`. Errors when `numbers` is empty.",
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

var _ function.Function = (*PercentileFunction)(nil)

type PercentileFunction struct{}

func NewPercentileFunction() function.Function { return &PercentileFunction{} }

func (f *PercentileFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "percentile"
}

func (f *PercentileFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Percentile of a list of numbers, by linear interpolation",
		MarkdownDescription: "Returns the `p`-th percentile of `numbers` using linear interpolation between adjacent ordered values. This is **Hyndman & Fan Type 7** — the default method in [NumPy](https://numpy.org/doc/stable/reference/generated/numpy.percentile.html), R, and Excel's `PERCENTILE.INC`.\n\nDefinition: let the sorted observations be `x[0] ≤ x[1] ≤ … ≤ x[N-1]`. Compute `h = (p / 100) × (N - 1)`. If `h` is an integer, return `x[h]`. Otherwise return `x[⌊h⌋] + (h - ⌊h⌋) × (x[⌈h⌉] - x[⌊h⌋])`.\n\nValid `p` is in `[0, 100]`. `p = 0` returns the minimum; `p = 100` returns the maximum; `p = 50` matches `median(numbers)`.",
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

	prec := statsPrec(xs)
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

	// h = (p / 100) * (N - 1)
	h := new(big.Float).SetPrec(prec).Quo(p, hundred)
	h.Mul(h, new(big.Float).SetPrec(prec).SetInt64(int64(n-1)))

	// h is non-negative (we validated p ≥ 0), so big.Float.Int's truncation toward zero coincides with floor.
	floorInt, _ := h.Int(nil)
	floorIdxF := new(big.Float).SetPrec(prec).SetInt(floorInt)
	frac := new(big.Float).SetPrec(prec).Sub(h, floorIdxF)
	idx := floorInt.Int64()

	// Exact integer index — no interpolation needed. Covers p = 0, p = 100, and any p that lands on an observation.
	if frac.Sign() == 0 {
		out := new(big.Float).SetPrec(prec).Copy(sorted[idx])
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, out))
		return
	}

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

var _ function.Function = (*VarianceFunction)(nil)

type VarianceFunction struct{}

func NewVarianceFunction() function.Function { return &VarianceFunction{} }

func (f *VarianceFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "variance"
}

func (f *VarianceFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Population variance of a list of numbers",
		MarkdownDescription: "Returns the **population variance** σ² = Σ (x − μ)² / N, where μ = `mean(numbers)`.\n\nThis is the population formula (divide by N), matching numpy's default. For sample variance (Bessel's correction, divide by N-1), multiply by `length(numbers) / (length(numbers) - 1)`. Errors when `numbers` is empty.",
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

var _ function.Function = (*StddevFunction)(nil)

type StddevFunction struct{}

func NewStddevFunction() function.Function { return &StddevFunction{} }

func (f *StddevFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "stddev"
}

func (f *StddevFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Population standard deviation of a list of numbers",
		MarkdownDescription: "Returns σ = √(Σ (x − μ)² / N), the **population standard deviation**, where μ = `mean(numbers)`.\n\nPopulation formula (divide by N), matching numpy's default. For sample standard deviation, take `sqrt(variance(numbers) × length(numbers) / (length(numbers) - 1))`. Errors when `numbers` is empty.",
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

var _ function.Function = (*ModeFunction)(nil)

type ModeFunction struct{}

func NewModeFunction() function.Function { return &ModeFunction{} }

func (f *ModeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "mode"
}

func (f *ModeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Most-frequent value(s) in a list of numbers",
		MarkdownDescription: "Returns the value(s) appearing most frequently in `numbers`, as a sorted ascending list. The result is always a list because the data may be **multimodal** — e.g. `mode([1, 1, 2, 2, 3])` is `[1, 2]`, not just one of them. For unimodal data the list has length 1.\n\nTwo numeric values are considered equal here when they compare equal as `*big.Float` (`Cmp == 0`), so `mode([1, 1.0])` collapses to `[1]`. Errors when `numbers` is empty.",
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
	for _, b := range buckets {
		if b.count == maxCount {
			modes = append(modes, b.value)
		}
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, modes))
}
