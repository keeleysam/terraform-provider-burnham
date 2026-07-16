package numerics

import (
	"context"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// runPercentile invokes PercentileFunction.Run in-process with the given
// observations and p, returning the result as a *big.Float.
func runPercentile(t *testing.T, xs []float64, p float64) (*big.Float, *function.FuncError) {
	t.Helper()
	elems := make([]attr.Value, len(xs))
	for i, v := range xs {
		elems[i] = types.NumberValue(big.NewFloat(v))
	}
	listVal, diags := types.ListValue(types.NumberType, elems)
	if diags.HasError() {
		t.Fatalf("building list value: %v", diags)
	}
	pVal := types.NumberValue(big.NewFloat(p))
	args := function.NewArgumentsData([]attr.Value{listVal, pVal})
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.NumberValue(new(big.Float)))}
	(&PercentileFunction{}).Run(context.Background(), req, resp)
	if resp.Error != nil {
		return nil, resp.Error
	}
	result, ok := resp.Result.Value().(types.Number)
	if !ok {
		t.Fatalf("expected Number result, got %T", resp.Result.Value())
	}
	return result.ValueBigFloat(), nil
}

func TestPercentile_ExactIndexFastPath(t *testing.T) {
	// percentile([0..25], 4): h = (4/100) * 25 = 1 exactly, so the answer is
	// x[1] = 1 exactly. The index lands on an observation, so no interpolation
	// (and no binary rounding of 0.04) should leak into the result.
	xs := make([]float64, 26)
	for i := range xs {
		xs[i] = float64(i)
	}
	got, err := runPercentile(t, xs, 4)
	if err != nil {
		t.Fatalf("percentile errored: %s", err.Text)
	}
	if got.Cmp(big.NewFloat(1)) != 0 {
		t.Errorf("percentile([0..25], 4) = %s, want exactly 1", got.Text('g', -1))
	}
}

func TestPercentile_LinearInterpolationSpotCheck(t *testing.T) {
	// numpy: percentile([10, 20], 30) == 13. h = 0.3 * 1 = 0.3 lands between
	// observations, so we interpolate: 10 + 0.3 * (20 - 10) = 13.
	got, err := runPercentile(t, []float64{10, 20}, 30)
	if err != nil {
		t.Fatalf("percentile errored: %s", err.Text)
	}
	diff := new(big.Float).Sub(got, big.NewFloat(13))
	diff.Abs(diff)
	if diff.Cmp(big.NewFloat(1e-12)) > 0 {
		t.Errorf("percentile([10, 20], 30) = %s, want ≈ 13", got.Text('g', -1))
	}
}
