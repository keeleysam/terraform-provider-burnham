package numerics

import (
	"context"
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// runPiApproximateDigit invokes pi_approximate_digit with n encoded as a types.NumberValue so we can pass arbitrarily-large *big.Float values.
func runPiApproximateDigit(t *testing.T, n *big.Float) (string, *function.FuncError) {
	t.Helper()
	f := &PiApproximateDigitFunction{}
	args := function.NewArgumentsData([]attr.Value{types.NumberValue(n)})
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.StringValue(""))}
	f.Run(context.Background(), req, resp)
	if resp.Error != nil {
		return "", resp.Error
	}
	result, ok := resp.Result.Value().(types.String)
	if !ok {
		t.Fatalf("expected String result, got %T", resp.Result.Value())
	}
	return result.ValueString(), nil
}

// runPiApproximateDigitInt is a convenience wrapper for int64 inputs.
func runPiApproximateDigitInt(t *testing.T, n int64) (string, *function.FuncError) {
	return runPiApproximateDigit(t, new(big.Float).SetInt64(n))
}

func runPiApproximateDigits(t *testing.T, count int64) (string, *function.FuncError) {
	t.Helper()
	f := &PiApproximateDigitsFunction{}
	args := function.NewArgumentsData([]attr.Value{types.Int64Value(count)})
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.StringValue(""))}
	f.Run(context.Background(), req, resp)
	if resp.Error != nil {
		return "", resp.Error
	}
	result, ok := resp.Result.Value().(types.String)
	if !ok {
		t.Fatalf("expected String result, got %T", resp.Result.Value())
	}
	return result.ValueString(), nil
}

func TestPiApproximateDigit_OneCycleByValue(t *testing.T) {
	// First six digits of 22/7 after the decimal: "142857".
	expected := "142857"
	for i := int64(1); i <= 6; i++ {
		got, err := runPiApproximateDigitInt(t, i)
		if err != nil {
			t.Errorf("pi_approximate_digit(%d) errored: %s", i, err.Text)
			continue
		}
		want := []byte{}
		want = append(want, []byte(formatInt(i))...)
		want = append(want, ':', expected[i-1])
		if got != string(want) {
			t.Errorf("pi_approximate_digit(%d) = %q, want %q", i, got, string(want))
		}
	}
}

func TestPiApproximateDigit_CycleWrap(t *testing.T) {
	// digit 7 wraps back to the first digit of "142857".
	got, err := runPiApproximateDigitInt(t, 7)
	if err != nil {
		t.Fatal(err.Text)
	}
	if got != "7:1" {
		t.Errorf("pi_approximate_digit(7) = %q, want \"7:1\"", got)
	}
}

func TestPiApproximateDigit_RejectsZero(t *testing.T) {
	_, err := runPiApproximateDigitInt(t, 0)
	if err == nil {
		t.Fatal("pi_approximate_digit(0) should error")
	}
	if !strings.Contains(err.Text, "n >= 1") {
		t.Errorf("error message = %q, expected mention of 'n >= 1'", err.Text)
	}
}

func TestPiApproximateDigit_RejectsNegative(t *testing.T) {
	_, err := runPiApproximateDigitInt(t, -1)
	if err == nil {
		t.Fatal("pi_approximate_digit(-1) should error")
	}
}

func TestPiApproximateDigit_RejectsFractional(t *testing.T) {
	half := big.NewFloat(0.5)
	_, err := runPiApproximateDigit(t, half)
	if err == nil {
		t.Fatal("pi_approximate_digit(0.5) should error")
	}
	if !strings.Contains(err.Text, "whole number") {
		t.Errorf("error message = %q, expected mention of 'whole number'", err.Text)
	}
}

func TestPiApproximateDigit_AtMaxInt64(t *testing.T) {
	// math.MaxInt64 - 1 is divisible by 6 with no remainder, so the digit
	// at math.MaxInt64 is at offset 0 → "1".
	// Verify computationally: (MaxInt64 - 1) % 6 = ?
	//   MaxInt64 = 9223372036854775807
	//   MaxInt64 - 1 = 9223372036854775806
	//   9223372036854775806 mod 6: 9223372036854775806 / 6 = 1537228672809129301 remainder 0
	//   So the digit is "142857"[0] = '1'.
	got, err := runPiApproximateDigitInt(t, math.MaxInt64)
	if err != nil {
		t.Fatalf("pi_approximate_digit(MaxInt64) errored: %s", err.Text)
	}
	want := "9223372036854775807:1"
	if got != want {
		t.Errorf("pi_approximate_digit(MaxInt64) = %q, want %q", got, want)
	}
}

// bigFloatExactInt builds a *big.Float that exactly represents the given big.Int. Default-precision big.Float (53 bits) can't hold integers above ~2^53, so we explicitly request precision wide enough for the value's bit length.
func bigFloatExactInt(i *big.Int) *big.Float {
	prec := uint(i.BitLen() + 64)
	if prec < 64 {
		prec = 64
	}
	return new(big.Float).SetPrec(prec).SetInt(i)
}

func TestPiApproximateDigit_PastInt64(t *testing.T) {
	// 10^30 — beyond int64 range (~9.2e18). Forces the math/big path.
	// (10^30 - 1) mod 6: 10^k mod 6 = 4 for all k >= 1, so 10^30 mod 6 = 4
	// and (10^30 - 1) mod 6 = 3. digit = "142857"[3] = '8'.
	ten30 := new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil)
	got, err := runPiApproximateDigit(t, bigFloatExactInt(ten30))
	if err != nil {
		t.Fatalf("pi_approximate_digit(1e30) errored: %s", err.Text)
	}
	const want = "1000000000000000000000000000000:8"
	if got != want {
		t.Errorf("pi_approximate_digit(1e30) = %q, want %q", got, want)
	}
}

func TestPiApproximateDigit_NearTerraformCeiling(t *testing.T) {
	// 10^150 — pushes near Terraform's 512-bit number ceiling.
	// (10^150 - 1) mod 6: same argument as above — every 10^k mod 6 = 4
	// for k >= 1, so 10^150 - 1 mod 6 = 3. Expected digit: '8'.
	ten150 := new(big.Int).Exp(big.NewInt(10), big.NewInt(150), nil)
	got, err := runPiApproximateDigit(t, bigFloatExactInt(ten150))
	if err != nil {
		t.Fatalf("pi_approximate_digit(1e150) errored: %s", err.Text)
	}
	want := "1" + strings.Repeat("0", 150) + ":8"
	if got != want {
		t.Errorf("pi_approximate_digit(1e150) = %q, want %q", got, want)
	}
}

func TestPiApproximateDigits_BasicCycles(t *testing.T) {
	cases := []struct {
		count int64
		want  string
	}{
		{0, ""},
		{1, "1"},
		{6, "142857"},
		{12, "142857142857"},
		{15, "142857142857142"},
	}
	for _, c := range cases {
		got, err := runPiApproximateDigits(t, c.count)
		if err != nil {
			t.Errorf("pi_approximate_digits(%d) errored: %s", c.count, err.Text)
			continue
		}
		if got != c.want {
			t.Errorf("pi_approximate_digits(%d) = %q, want %q", c.count, got, c.want)
		}
	}
}

func TestPiApproximateDigits_LargeCount(t *testing.T) {
	got, err := runPiApproximateDigits(t, 100_000)
	if err != nil {
		t.Fatalf("pi_approximate_digits(100000) errored: %s", err.Text)
	}
	if len(got) != 100_000 {
		t.Errorf("length = %d, want 100000", len(got))
	}
	// Must consist of the cycle "142857" repeated.
	for i := 0; i < len(got); i++ {
		if got[i] != approximateCycle[i%6] {
			t.Errorf("digit[%d] = %q, want %q", i, got[i], approximateCycle[i%6])
			break
		}
	}
}

func TestPiApproximateDigits_RejectsNegative(t *testing.T) {
	_, err := runPiApproximateDigits(t, -1)
	if err == nil {
		t.Fatal("pi_approximate_digits(-1) should error")
	}
}

// formatInt is a tiny helper to avoid pulling in strconv just here.
func formatInt(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
