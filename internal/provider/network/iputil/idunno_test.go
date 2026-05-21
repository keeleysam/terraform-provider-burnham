package iputil

import (
	"net/netip"
	"strings"
	"testing"
	"unicode/utf8"
)

// TestIDunnoEncodeRFC5Example locks the RFC 8771 §5 worked-example vector:
// "198.51.100.164" must produce exactly the four codepoints the RFC names —
// U+0063 LATIN SMALL LETTER C, U+000C FORM FEED, U+006C LATIN SMALL LETTER L,
// U+04A4 CYRILLIC CAPITAL LIGATURE EN GHE — using layout 7+7+7+11. Any
// change to the encoder's layout-priority order that breaks this vector is a
// breaking change to the RFC's only published worked example, so this test is
// the one to be loudest about a regression.
func TestIDunnoEncodeRFC5Example(t *testing.T) {
	got, err := IDunnoEncode("198.51.100.164")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	want := string([]rune{0x0063, 0x000C, 0x006C, 0x04A4})
	if got != want {
		t.Errorf("RFC §5 worked example regression:\n  got:  %q (codepoints %s)\n  want: %q (codepoints %s)", got, codepointHex(got), want, codepointHex(want))
	}
}

// TestIDunnoEncodeAllValid sweeps a representative range of IPv4 and IPv6
// addresses (including all-zero, all-one, alternating, common reserved blocks)
// and confirms the encoder:
//   - Produces a non-empty string
//   - Produces only valid UTF-8
//   - Reaches Minimum Confusion Level per §4.1
//   - Round-trips through IDunnoDecode back to the input
func TestIDunnoEncodeAllValid(t *testing.T) {
	cases := []string{
		// IPv4
		"0.0.0.0", "255.255.255.255", "127.0.0.1", "10.0.0.1",
		"192.168.1.1", "203.0.113.7", "198.51.100.164",
		"169.254.1.1", "224.0.0.1", "8.8.8.8", "1.1.1.1",
		"100.64.0.1", "172.16.0.1", "239.255.255.255",
		// IPv4 boundary nibbles (last 4 bits = 0 hits the 7+7+7+11 fallback path)
		"1.2.3.0", "1.2.3.16", "1.2.3.32", "1.2.3.240",
		// IPv6
		"::", "::1", "::ffff:1.2.3.4",
		"2001:db8::1", "fe80::1", "ff02::1", "fc00::1",
		"2001:db8:1234:5678:9abc:def0:1234:5678",
		"ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff",
		"::1234:5678", "1::", "0:0:0:0:0:0:0:0",
	}
	for _, ip := range cases {
		t.Run(ip, func(t *testing.T) {
			got, err := IDunnoEncode(ip)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			if got == "" {
				t.Fatal("encoder returned empty string")
			}
			if !utf8.ValidString(got) {
				t.Fatalf("encoder returned invalid UTF-8: %q", got)
			}
			cps := []rune(got)
			if !idunnoSatisfiesMinCL(cps) {
				t.Errorf("RFC §4.1 Min CL not reached: codepoints %s", codepointHex(got))
			}
			// Round-trip.
			back, err := IDunnoDecode(got)
			if err != nil {
				t.Fatalf("decode %q: %v", got, err)
			}
			// Canonicalise the input the same way the decoder does so the
			// equality check ignores cosmetic differences like ::ffff:1.2.3.4
			// → 1.2.3.4 (Unmap) or 1::/::1 vs explicit colon-hex form.
			wantAddr, _ := netip.ParseAddr(ip)
			wantStr := wantAddr.Unmap().String()
			if back != wantStr {
				t.Errorf("round-trip mismatch: input %q → encoded %q → decoded %q (want %q)", ip, got, back, wantStr)
			}
		})
	}
}

// TestIDunnoDecodeMalformed exercises the decoder's error paths.
func TestIDunnoDecodeMalformed(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"invalid UTF-8", "\xff\xfe\xfd"},
		// 3 codepoints × 7 bits each = 21 bits; not in any address-bit range.
		{"too-short payload", "abc"},
		// 100 ASCII chars × 7 bits = 700 bits; way over the IPv6 cap.
		{"too-long payload", strings.Repeat("a", 100)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := IDunnoDecode(c.in); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

// TestIDunnoEncodeRejectsInvalid confirms the encoder gives a useful error
// for things that aren't IP addresses.
func TestIDunnoEncodeRejectsInvalid(t *testing.T) {
	cases := []string{"", "not-an-ip", "256.256.256.256", "2001:db8:::1", "1.2.3.4/24"}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			if _, err := IDunnoEncode(c); err == nil {
				t.Fatalf("expected error for %q, got nil", c)
			}
		})
	}
}

// TestIDunnoCodepointWidth locks the UTF-8-length → bit-count mapping from RFC
// §3 Table 1. Any change to idunnoBitsForCodepoint is a wire-format change.
func TestIDunnoCodepointWidth(t *testing.T) {
	cases := []struct {
		cp        rune
		wantBits  int
		wantError bool
	}{
		{0x0000, 7, false}, {0x007F, 7, false},
		{0x0080, 11, false}, {0x07FF, 11, false},
		{0x0800, 16, false}, {0xD7FF, 16, false},
		{0xD800, 0, true}, {0xDFFF, 0, true}, // surrogates
		{0xE000, 16, false}, {0xFFFF, 16, false},
		{0x10000, 21, false}, {0x10FFFF, 21, false},
		{0x110000, 0, true}, // above max valid codepoint
		{-1, 0, true},
	}
	for _, c := range cases {
		got, err := idunnoBitsForCodepoint(c.cp)
		if c.wantError {
			if err == nil {
				t.Errorf("U+%04X: expected error, got bits=%d", c.cp, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("U+%04X: unexpected error: %v", c.cp, err)
			continue
		}
		if got != c.wantBits {
			t.Errorf("U+%04X: got %d bits, want %d", c.cp, got, c.wantBits)
		}
	}
}

// TestIDunnoValidCP locks the per-chunk-size codepoint validity check.
func TestIDunnoValidCP(t *testing.T) {
	type c struct {
		value    int
		size     int
		want     bool
		whatItIs string
	}
	cases := []c{
		// 7-bit (1-byte UTF-8): 0x00–0x7F.
		{0x00, 7, true, "NUL"},
		{0x7F, 7, true, "DEL"},
		{0x80, 7, false, "overflow into 2-byte range"},
		// 11-bit (2-byte UTF-8): 0x80–0x7FF (rejects overlong 0x00–0x7F).
		{0x7F, 11, false, "overlong"},
		{0x80, 11, true, "first valid 2-byte"},
		{0x7FF, 11, true, "last 2-byte"},
		{0x800, 11, false, "into 3-byte range"},
		// 16-bit (3-byte UTF-8): 0x800–0xFFFF excluding surrogates.
		{0x800, 16, true, "first 3-byte"},
		{0xD7FF, 16, true, "just below surrogate"},
		{0xD800, 16, false, "surrogate lo"},
		{0xDFFF, 16, false, "surrogate hi"},
		{0xE000, 16, true, "just above surrogate"},
		{0xFFFF, 16, true, "last 3-byte"},
		{0x10000, 16, false, "into 4-byte range"},
		// 21-bit (4-byte UTF-8): 0x10000–0x10FFFF.
		{0xFFFF, 21, false, "below 4-byte"},
		{0x10000, 21, true, "first 4-byte"},
		{0x10FFFF, 21, true, "last valid codepoint"},
		{0x110000, 21, false, "above max Unicode"},
	}
	for _, tc := range cases {
		got := idunnoValidCP(tc.value, tc.size)
		if got != tc.want {
			t.Errorf("idunnoValidCP(%#X, %d) [%s] = %v, want %v", tc.value, tc.size, tc.whatItIs, got, tc.want)
		}
	}
}

// TestIDunnoMinCL verifies the Min-Confusion-Level checker fires correctly.
func TestIDunnoMinCL(t *testing.T) {
	cases := []struct {
		name string
		cps  []rune
		want bool
	}{
		{"RFC §5 example", []rune{0x0063, 0x000C, 0x006C, 0x04A4}, true}, // U+000C is C0 control, U+04A4 is multi-octet
		{"all 7-bit lowercase", []rune{'a', 'b', 'c', 'd'}, false},       // no multi-octet
		{"multi-octet but no disallowed", []rune{0x0800, 0x0801}, false}, // U+0800/0x0801 are Samaritan letters (PVALID)
		{"control + multi-octet", []rune{0x0001, 0x0800}, true},          // U+0001 is C0 control
		{"ASCII upper + multi-octet", []rune{'A', 0x0800}, true},         // 'A' is MAPPED → counted disallowed
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := idunnoSatisfiesMinCL(tc.cps); got != tc.want {
				t.Errorf("idunnoSatisfiesMinCL = %v, want %v (codepoints %v)", got, tc.want, tc.cps)
			}
		})
	}
}

// codepointHex pretty-prints a string as a sequence of U+NNNN labels for test
// failure messages.
func codepointHex(s string) string {
	var sb strings.Builder
	for i, cp := range s {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString("U+")
		hex := []byte{0, 0, 0, 0}
		for j := 3; j >= 0; j-- {
			d := int(cp) & 0xF
			cp >>= 4
			hexDigit := byte('0' + d)
			if d > 9 {
				hexDigit = byte('A' + d - 10)
			}
			hex[j] = hexDigit
		}
		sb.Write(hex)
	}
	return sb.String()
}
