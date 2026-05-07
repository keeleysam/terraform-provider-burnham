package numerics

import (
	"math/big"
	"testing"
)

// BenchmarkPiDigitChar measures the raw single-digit lookup against the
// embedded DPD-packed table. This is what `pi_digit(n)` calls under the
// hood (plus a fmt.Sprintf for the "n:digit" framing).
func BenchmarkPiDigitChar(b *testing.B) {
	for _, n := range []int64{1, 100, 10_000, 100_000, 999_999, 3_141_592} {
		b.Run("n="+itoaB(n), func(b *testing.B) {
			b.ReportAllocs()
			var sink byte
			for i := 0; i < b.N; i++ {
				sink = piDigitChar(n)
			}
			_ = sink
		})
	}
}

// BenchmarkPiFirstNDigits measures the bulk-extract path used by
// `pi_digits(count)`. The work scales linearly with count.
func BenchmarkPiFirstNDigits(b *testing.B) {
	for _, n := range []int64{10, 100, 1_000, 10_000, 100_000, 1_000_000, 3_141_592} {
		b.Run("count="+itoaB(n), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(n)
			var sink string
			for i := 0; i < b.N; i++ {
				sink = piFirstNDigits(n)
			}
			_ = sink
		})
	}
}

// BenchmarkApproximateDigitChar measures the 22/7 single-digit lookup.
// Uses big.Int.Mod under the hood; we reuse the *big.Int across iterations
// so we measure just the modular-arithmetic cost.
func BenchmarkApproximateDigitChar(b *testing.B) {
	cases := []struct {
		name string
		n    *big.Int
	}{
		{"n=1", big.NewInt(1)},
		{"n=1e6", big.NewInt(1_000_000)},
		{"n=MaxInt64", new(big.Int).SetInt64(9_223_372_036_854_775_807)},
		{"n=1e30", new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil)},
		{"n=1e150", new(big.Int).Exp(big.NewInt(10), big.NewInt(150), nil)},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			var sink byte
			for i := 0; i < b.N; i++ {
				sink = approximateDigitChar(c.n)
			}
			_ = sink
		})
	}
}

// BenchmarkApproximateFirstNDigits measures bulk 22/7 generation. Pure
// modulo-based loop — no I/O, no big.Int arithmetic.
func BenchmarkApproximateFirstNDigits(b *testing.B) {
	for _, n := range []int64{10, 100, 1_000, 10_000, 100_000, 1_000_000, 3_141_592} {
		b.Run("count="+itoaB(n), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(n)
			var sink string
			for i := 0; i < b.N; i++ {
				sink = approximateFirstNDigits(n)
			}
			_ = sink
		})
	}
}

// itoaB is a tiny helper local to the benchmark file. Keeps benchmark
// names readable without pulling strconv just for this.
func itoaB(n int64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
