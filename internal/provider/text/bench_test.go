package text

import (
	"strings"
	"testing"

	"rsc.io/qr"
)

// benchString builds an n-rune ASCII string with word-level structure, deterministic across runs.
func benchString(n int) string {
	const seed = "the quick brown fox jumps over the lazy dog while terraform plans converge "
	var b strings.Builder
	b.Grow(n + len(seed))
	for b.Len() < n {
		b.WriteString(seed)
	}
	return b.String()[:n]
}

// BenchmarkLevenshtein is O(len(a) * len(b)); the interesting axis is string length.
func BenchmarkLevenshtein(b *testing.B) {
	for _, n := range []int{64, 512, 2048} {
		a := benchString(n)
		// A second string that differs throughout, same length.
		c := strings.ReplaceAll(a, "o", "0")
		b.Run(itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = levenshteinDistance(a, c)
			}
		})
	}
}

// BenchmarkQR covers the closest existing "generate a visual artifact" function:
// encode (rsc.io/qr) plus the half-block render.
func BenchmarkQR(b *testing.B) {
	cases := []struct {
		name    string
		payload string
	}{
		{"url", "https://registry.terraform.io/providers/keeleysam/burnham/latest"},
		{"1KB", benchString(1024)},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				code, err := qr.Encode(tc.payload, qr.M)
				if err != nil {
					b.Fatal(err)
				}
				_ = renderHalfBlock(code, 4, false)
			}
		})
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [16]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
