package numerics

import (
	"context"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// numberVal builds a types.Number from an integer without going through
// float64, so large values (2^64, 2^100) survive intact.
func numberVal(i *big.Int) types.Number {
	return types.NumberValue(new(big.Float).SetInt(i))
}

// runListFn invokes a folded list-of-number bitwise function in-process.
func runListFn(t *testing.T, fn function.Function, floats []*big.Float) (*big.Int, *function.FuncError) {
	t.Helper()
	elems := make([]attr.Value, len(floats))
	for i, v := range floats {
		elems[i] = types.NumberValue(v)
	}
	lv, diags := types.ListValue(types.NumberType, elems)
	if diags.HasError() {
		t.Fatalf("building list: %v", diags)
	}
	args := function.NewArgumentsData([]attr.Value{lv})
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.NumberValue(new(big.Float)))}
	fn.Run(context.Background(), req, resp)
	if resp.Error != nil {
		return nil, resp.Error
	}
	num, ok := resp.Result.Value().(types.Number)
	if !ok {
		t.Fatalf("expected Number, got %T", resp.Result.Value())
	}
	out, _ := num.ValueBigFloat().Int(nil)
	return out, nil
}

// runNumberFn invokes a bitwise function taking integer args, returning a Number.
func runNumberFn(t *testing.T, fn function.Function, ints ...*big.Int) (*big.Int, *function.FuncError) {
	t.Helper()
	vals := make([]attr.Value, len(ints))
	for i, v := range ints {
		vals[i] = numberVal(v)
	}
	args := function.NewArgumentsData(vals)
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.NumberValue(new(big.Float)))}
	fn.Run(context.Background(), req, resp)
	if resp.Error != nil {
		return nil, resp.Error
	}
	num, ok := resp.Result.Value().(types.Number)
	if !ok {
		t.Fatalf("expected Number, got %T", resp.Result.Value())
	}
	out, _ := num.ValueBigFloat().Int(nil)
	return out, nil
}

// runBoolFn invokes a bitwise function returning a bool.
func runBoolFn(t *testing.T, fn function.Function, ints ...*big.Int) (bool, *function.FuncError) {
	t.Helper()
	vals := make([]attr.Value, len(ints))
	for i, v := range ints {
		vals[i] = numberVal(v)
	}
	args := function.NewArgumentsData(vals)
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.BoolValue(false))}
	fn.Run(context.Background(), req, resp)
	if resp.Error != nil {
		return false, resp.Error
	}
	b, ok := resp.Result.Value().(types.Bool)
	if !ok {
		t.Fatalf("expected Bool, got %T", resp.Result.Value())
	}
	return b.ValueBool(), nil
}

func bf(i int64) *big.Float { return new(big.Float).SetInt64(i) }
func bi(i int64) *big.Int   { return big.NewInt(i) }

func wantInt(t *testing.T, got *big.Int, ferr *function.FuncError, want int64) {
	t.Helper()
	if ferr != nil {
		t.Fatalf("unexpected error: %s", ferr.Text)
	}
	if got.Cmp(big.NewInt(want)) != 0 {
		t.Errorf("got %s, want %d", got.String(), want)
	}
}

// ── folded list functions ─────────────────────────────────────────────

func TestBitAnd(t *testing.T) {
	got, ferr := runListFn(t, &BitAndFunction{}, []*big.Float{bf(12), bf(10)})
	wantInt(t, got, ferr, 8)
}

func TestBitOr(t *testing.T) {
	got, ferr := runListFn(t, &BitOrFunction{}, []*big.Float{bf(1), bf(2), bf(8)})
	wantInt(t, got, ferr, 11)
}

func TestBitXor(t *testing.T) {
	got, ferr := runListFn(t, &BitXorFunction{}, []*big.Float{bf(5), bf(3)})
	wantInt(t, got, ferr, 6)

	got, ferr = runListFn(t, &BitXorFunction{}, []*big.Float{bf(5), bf(3), bf(6)})
	wantInt(t, got, ferr, 0)
}

func TestBitAnd_SingleElement(t *testing.T) {
	got, ferr := runListFn(t, &BitAndFunction{}, []*big.Float{bf(42)})
	wantInt(t, got, ferr, 42)
}

func TestBitFold_EmptyListErrors(t *testing.T) {
	if _, ferr := runListFn(t, &BitAndFunction{}, []*big.Float{}); ferr == nil {
		t.Fatal("expected error for empty list")
	}
}

func TestBitFold_NonIntegerErrors(t *testing.T) {
	if _, ferr := runListFn(t, &BitAndFunction{}, []*big.Float{big.NewFloat(1.5), bf(2)}); ferr == nil {
		t.Fatal("expected error for non-integer element")
	}
}

// ── bit_not ───────────────────────────────────────────────────────────

func TestBitNot(t *testing.T) {
	got, ferr := runNumberFn(t, &BitNotFunction{}, bi(0), bi(8))
	wantInt(t, got, ferr, 255)

	got, ferr = runNumberFn(t, &BitNotFunction{}, bi(255), bi(8))
	wantInt(t, got, ferr, 0)

	got, ferr = runNumberFn(t, &BitNotFunction{}, bi(1), bi(4))
	wantInt(t, got, ferr, 14)
}

func TestBitNot_OutOfRangeErrors(t *testing.T) {
	if _, ferr := runNumberFn(t, &BitNotFunction{}, bi(256), bi(8)); ferr == nil {
		t.Fatal("expected error for value >= 2^bits")
	}
	if _, ferr := runNumberFn(t, &BitNotFunction{}, bi(-1), bi(8)); ferr == nil {
		t.Fatal("expected error for negative value")
	}
}

func TestBitNot_BitsTooSmallErrors(t *testing.T) {
	if _, ferr := runNumberFn(t, &BitNotFunction{}, bi(0), bi(0)); ferr == nil {
		t.Fatal("expected error for bits < 1")
	}
}

// ── shifts ────────────────────────────────────────────────────────────

func TestBitShiftLeft(t *testing.T) {
	got, ferr := runNumberFn(t, &BitShiftLeftFunction{}, bi(1), bi(10))
	wantInt(t, got, ferr, 1024)
}

func TestBitShiftLeft_BigIntBeyondInt64(t *testing.T) {
	got, ferr := runNumberFn(t, &BitShiftLeftFunction{}, bi(1), bi(100))
	if ferr != nil {
		t.Fatalf("unexpected error: %s", ferr.Text)
	}
	want := new(big.Int).Lsh(big.NewInt(1), 100)
	if got.Cmp(want) != 0 {
		t.Errorf("got %s, want %s", got.String(), want.String())
	}
}

func TestBitShiftRight(t *testing.T) {
	got, ferr := runNumberFn(t, &BitShiftRightFunction{}, bi(1024), bi(3))
	wantInt(t, got, ferr, 128)
}

func TestBitShiftRight_ArithmeticFloorsNegative(t *testing.T) {
	// -8 >> 1 floors toward negative infinity: -4. -1 >> 1 = -1.
	got, ferr := runNumberFn(t, &BitShiftRightFunction{}, bi(-8), bi(1))
	wantInt(t, got, ferr, -4)

	got, ferr = runNumberFn(t, &BitShiftRightFunction{}, bi(-1), bi(1))
	wantInt(t, got, ferr, -1)
}

func TestShift_NegativeNErrors(t *testing.T) {
	if _, ferr := runNumberFn(t, &BitShiftLeftFunction{}, bi(1), bi(-1)); ferr == nil {
		t.Fatal("expected error for negative shift")
	}
	if _, ferr := runNumberFn(t, &BitShiftRightFunction{}, bi(1), bi(-1)); ferr == nil {
		t.Fatal("expected error for negative shift")
	}
}

// ── popcount ──────────────────────────────────────────────────────────

func TestPopcount(t *testing.T) {
	got, ferr := runNumberFn(t, &PopcountFunction{}, bi(255))
	wantInt(t, got, ferr, 8)

	got, ferr = runNumberFn(t, &PopcountFunction{}, bi(0))
	wantInt(t, got, ferr, 0)
}

func TestPopcount_BeyondInt64(t *testing.T) {
	// 2^64 has exactly one set bit.
	twoTo64 := new(big.Int).Lsh(big.NewInt(1), 64)
	got, ferr := runNumberFn(t, &PopcountFunction{}, twoTo64)
	wantInt(t, got, ferr, 1)

	// 2^100 - 1 has exactly 100 set bits.
	allOnes := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 100), big.NewInt(1))
	got, ferr = runNumberFn(t, &PopcountFunction{}, allOnes)
	wantInt(t, got, ferr, 100)
}

func TestPopcount_NegativeErrors(t *testing.T) {
	if _, ferr := runNumberFn(t, &PopcountFunction{}, bi(-1)); ferr == nil {
		t.Fatal("expected error for negative value")
	}
}

// ── single-bit helpers ────────────────────────────────────────────────

func TestBitSet(t *testing.T) {
	got, ferr := runNumberFn(t, &BitSetFunction{}, bi(0), bi(3))
	wantInt(t, got, ferr, 8)

	// Setting an already-set bit is a no-op.
	got, ferr = runNumberFn(t, &BitSetFunction{}, bi(8), bi(3))
	wantInt(t, got, ferr, 8)
}

func TestBitClear(t *testing.T) {
	got, ferr := runNumberFn(t, &BitClearFunction{}, bi(15), bi(1))
	wantInt(t, got, ferr, 13)

	// Clearing an already-clear bit is a no-op.
	got, ferr = runNumberFn(t, &BitClearFunction{}, bi(13), bi(1))
	wantInt(t, got, ferr, 13)
}

func TestBitTest(t *testing.T) {
	got, ferr := runBoolFn(t, &BitTestFunction{}, bi(8), bi(3))
	if ferr != nil {
		t.Fatalf("unexpected error: %s", ferr.Text)
	}
	if !got {
		t.Error("bit_test(8, 3) = false, want true")
	}

	got, ferr = runBoolFn(t, &BitTestFunction{}, bi(8), bi(0))
	if ferr != nil {
		t.Fatalf("unexpected error: %s", ferr.Text)
	}
	if got {
		t.Error("bit_test(8, 0) = true, want false")
	}
}

func TestSingleBit_NegativeIndexErrors(t *testing.T) {
	if _, ferr := runNumberFn(t, &BitSetFunction{}, bi(0), bi(-1)); ferr == nil {
		t.Fatal("expected error for negative index")
	}
	if _, ferr := runNumberFn(t, &BitClearFunction{}, bi(0), bi(-1)); ferr == nil {
		t.Fatal("expected error for negative index")
	}
	if _, ferr := runBoolFn(t, &BitTestFunction{}, bi(0), bi(-1)); ferr == nil {
		t.Fatal("expected error for negative index")
	}
}
