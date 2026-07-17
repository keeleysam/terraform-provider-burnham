package numerics

import (
	"context"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// runNumberListFn invokes a single-list-of-number function in-process and
// returns the result as a *big.Float.
func runNumberListFn(t *testing.T, fn function.Function, xs []*big.Float) (*big.Float, *function.FuncError) {
	t.Helper()
	elems := make([]attr.Value, len(xs))
	for i, v := range xs {
		elems[i] = types.NumberValue(v)
	}
	listVal, diags := types.ListValue(types.NumberType, elems)
	if diags.HasError() {
		t.Fatalf("building list value: %v", diags)
	}
	args := function.NewArgumentsData([]attr.Value{listVal})
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.NumberValue(new(big.Float)))}
	fn.Run(context.Background(), req, resp)
	if resp.Error != nil {
		return nil, resp.Error
	}
	result, ok := resp.Result.Value().(types.Number)
	if !ok {
		t.Fatalf("expected Number result, got %T", resp.Result.Value())
	}
	return result.ValueBigFloat(), nil
}

// bigFloats builds a slice of *big.Float from int64 values.
func bigFloats(vals ...int64) []*big.Float {
	out := make([]*big.Float, len(vals))
	for i, v := range vals {
		out[i] = new(big.Float).SetInt64(v)
	}
	return out
}

func TestGCD(t *testing.T) {
	cases := []struct {
		name string
		in   []*big.Float
		want int64
	}{
		{"two values", bigFloats(12, 18), 6},
		{"three values", bigFloats(12, 18, 30), 6},
		{"single value", bigFloats(17), 17},
		{"negative operand", bigFloats(-12, 18), 6},
		{"all zeros", bigFloats(0, 0), 0},
		{"zero and n", bigFloats(0, 5), 5},
		{"both negative", bigFloats(-24, -36), 12},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := runNumberListFn(t, &GCDFunction{}, tc.in)
			if err != nil {
				t.Fatalf("gcd errored: %s", err.Text)
			}
			if got.Cmp(new(big.Float).SetInt64(tc.want)) != 0 {
				t.Errorf("gcd = %s, want %d", got.Text('g', -1), tc.want)
			}
		})
	}
}

func TestLCM(t *testing.T) {
	cases := []struct {
		name string
		in   []*big.Float
		want int64
	}{
		{"two values", bigFloats(4, 6), 12},
		{"three values", bigFloats(2, 3, 4), 12},
		{"single value", bigFloats(7), 7},
		{"zero present", bigFloats(0, 5), 0},
		{"negative operand", bigFloats(-4, 6), 12},
		{"single negative", bigFloats(-7), 7},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := runNumberListFn(t, &LCMFunction{}, tc.in)
			if err != nil {
				t.Fatalf("lcm errored: %s", err.Text)
			}
			if got.Cmp(new(big.Float).SetInt64(tc.want)) != 0 {
				t.Errorf("lcm = %s, want %d", got.Text('g', -1), tc.want)
			}
		})
	}
}

func TestLCMArbitraryPrecision(t *testing.T) {
	// Two coprime integers well beyond int64 (distinct prime powers): 2^70 and
	// 3^45. gcd is 1, so lcm is exactly their product. Proves the pairwise
	// a/gcd*b computation stays exact in arbitrary precision.
	a := new(big.Int).Exp(big.NewInt(2), big.NewInt(70), nil)
	b := new(big.Int).Exp(big.NewInt(3), big.NewInt(45), nil)
	want := new(big.Int).Mul(a, b)

	xs := []*big.Float{new(big.Float).SetInt(a), new(big.Float).SetInt(b)}
	got, err := runNumberListFn(t, &LCMFunction{}, xs)
	if err != nil {
		t.Fatalf("lcm errored: %s", err.Text)
	}
	if got.Cmp(new(big.Float).SetInt(want)) != 0 {
		t.Errorf("lcm(2^70, 3^45) = %s, want %s (exact product)", got.Text('g', -1), want.String())
	}
}

func TestGCDArbitraryPrecision(t *testing.T) {
	// gcd(2^70 * 3, 2^70 * 5) = 2^70, a value larger than int64.
	base := new(big.Int).Exp(big.NewInt(2), big.NewInt(70), nil)
	a := new(big.Int).Mul(base, big.NewInt(3))
	b := new(big.Int).Mul(base, big.NewInt(5))

	xs := []*big.Float{new(big.Float).SetInt(a), new(big.Float).SetInt(b)}
	got, err := runNumberListFn(t, &GCDFunction{}, xs)
	if err != nil {
		t.Fatalf("gcd errored: %s", err.Text)
	}
	if got.Cmp(new(big.Float).SetInt(base)) != 0 {
		t.Errorf("gcd = %s, want %s", got.Text('g', -1), base.String())
	}
}

func TestGCDNonIntegerErrors(t *testing.T) {
	xs := []*big.Float{big.NewFloat(1.5), new(big.Float).SetInt64(2)}
	_, err := runNumberListFn(t, &GCDFunction{}, xs)
	if err == nil {
		t.Fatal("expected an error for a non-integer element, got nil")
	}
}

func TestLCMNonIntegerErrors(t *testing.T) {
	xs := []*big.Float{new(big.Float).SetInt64(2), big.NewFloat(2.5)}
	_, err := runNumberListFn(t, &LCMFunction{}, xs)
	if err == nil {
		t.Fatal("expected an error for a non-integer element, got nil")
	}
}

func TestGCDEmptyListErrors(t *testing.T) {
	_, err := runNumberListFn(t, &GCDFunction{}, nil)
	if err == nil {
		t.Fatal("expected an error for an empty list, got nil")
	}
}

func TestLCMEmptyListErrors(t *testing.T) {
	_, err := runNumberListFn(t, &LCMFunction{}, nil)
	if err == nil {
		t.Fatal("expected an error for an empty list, got nil")
	}
}
