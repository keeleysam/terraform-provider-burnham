/*
RFC 8771 — Internationalized Deliberately Unreadable Network Notation (I-DUNNO).

  - Bit-packing per §3 Table 1:
      1-byte UTF-8 sequence carries  7 bits
      2-byte UTF-8 sequence carries 11 bits
      3-byte UTF-8 sequence carries 16 bits
      4-byte UTF-8 sequence carries 21 bits
  - The bitstring is the address in network byte order: 32 bits for IPv4,
    128 bits for IPv6 (§3.1).
  - Bits are consumed left-to-right; the final UTF-8 sequence may carry up
    to 20 bits of trailing padding (§3.1).
  - The encoding must reach Minimum Confusion Level (§4.1): at least one
    multi-octet UTF-8 sequence AND at least one character that is DISALLOWED
    per IDNA2008 (RFC 5892).

§3.2 says deforming "is intentionally omitted" because "humans SHOULD NOT
attempt the process." The machines DO know how to do it — we walk codepoints
left-to-right, infer each codepoint's UTF-8 length from its numeric value,
take that many of the codepoint's low-order bits, concatenate, and lop off
the trailing padding. Length disambiguates IPv4 (32–52 bits total) from
IPv6 (128–148 bits total); those ranges don't overlap so the decoder doesn't
need a hint.

RFC §5's worked example (198.51.100.164 → U+0063, U+000C, U+006C, U+04A4)
is the primary golden test vector. To round-trip that exactly the encoder
must choose layout 7+7+7+11 first; that is the first entry in
idunnoIPv4Layouts below.
*/

package iputil

import (
	"fmt"
	"net/netip"
	"strings"
	"unicode"
	"unicode/utf8"
)

// idunnoMaxPadBits is the per-RFC §3.1 limit on padding the final UTF-8 sequence.
const idunnoMaxPadBits = 20

// idunnoIPv4Layouts is the deterministic priority list of UTF-8-length sequences
// the encoder tries for a 32-bit IPv4 address. Layout {7, 7, 7, 11} is RFC §5's
// worked-example layout and is listed first so 198.51.100.164 round-trips to the
// RFC example exactly. The last entry, {7, 7, 7, 7, 16}, is a universal fallback
// that always produces a valid encoding for any IPv4 address (4 ASCII chunks
// consume the first 28 bits, then a 3-byte UTF-8 sequence with 12 bits of
// padding lets the encoder pick any codepoint in U+0000–U+FFFF that's ≥ U+0800).
var idunnoIPv4Layouts = [][]int{
	{7, 7, 7, 11},    // 32 bits, RFC §5 example. Works when bits[21:32] ≥ 0x80.
	{16, 16},         // 32 bits, all 3-byte. Both chunks must be ≥ 0x800 and not surrogate.
	{21, 11},         // 32 bits. First chunk must be ≥ U+10000.
	{11, 21},         // 32 bits.
	{11, 11, 11, 7},  // 32 bits. All 11-bit chunks must be ≥ 0x80.
	{7, 11, 7, 7},    // 32 bits.
	{7, 7, 11, 7},    // 32 bits.
	{11, 7, 7, 7},    // 32 bits.
	{7, 7, 7, 7, 11}, // 39 bits, 7 pad. Works when last nibble != 0 (gives padding room).
	{7, 7, 7, 16},    // 37 bits, 5 pad. Works when bits[21:32] ≥ 0x40.
	{7, 7, 7, 21},    // 42 bits, 10 pad. Final chunk needs value in [0x10000, 0x10FFFF].
	{7, 7, 7, 7, 16}, // 44 bits, 12 pad. UNIVERSAL FALLBACK — always works.
}

// idunnoIPv6Layouts is the priority list for 128-bit IPv6 addresses. Layouts
// here use fewer codepoints than the IPv4 list since IPv6 has more bits to pack;
// the encoder prefers compact layouts (more 3- and 4-byte sequences) for
// stronger per-codepoint obfuscation. The last entry is a universal fallback:
// 18 × 7-bit ASCII chunks consume 126 of 128 address bits, then a 3-byte UTF-8
// sequence with 14 bits of padding covers the last 2 address bits + any
// valid BMP codepoint ≥ U+0800.
var idunnoIPv6Layouts = [][]int{
	{16, 16, 16, 16, 16, 16, 16, 16},                           // 128 bits, 8×3-byte
	{21, 16, 16, 16, 16, 16, 16, 11},                           // 128 bits
	{21, 21, 21, 21, 21, 21, 7},                                // 133 bits, 5 pad
	{21, 21, 21, 21, 21, 21, 11},                               // 137 bits, 9 pad
	{21, 21, 21, 21, 21, 16, 16},                               // 137 bits, 9 pad
	{11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 7},            // 128 bits
	{16, 16, 16, 16, 16, 16, 16, 7, 11},                        // 130 bits, 2 pad
	{16, 16, 16, 16, 16, 16, 16, 16, 7},                        // 135 bits, 7 pad
	{16, 16, 16, 16, 16, 16, 16, 16, 11},                       // 139 bits, 11 pad
	{16, 16, 16, 16, 16, 16, 16, 16, 16},                       // 144 bits, 16 pad
	{7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 11}, // 137 bits, 9 pad (long-prefix for sparse addresses)
	{7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 16}, // 142 bits, 14 pad — UNIVERSAL FALLBACK
}

// IDunnoEncode produces the I-DUNNO representation of an IPv4 or IPv6 address
// per RFC 8771. The output is a UTF-8 string of Unicode codepoints.
//
// The encoder is deterministic for a given input and is faithful to RFC §3 and
// §4.1 (Minimum Confusion Level). RFC §5's example address (198.51.100.164)
// produces exactly U+0063, U+000C, U+006C, U+04A4.
func IDunnoEncode(ipStr string) (string, error) {
	addr, err := netip.ParseAddr(ipStr)
	if err != nil {
		return "", fmt.Errorf("invalid IP address %q: %w", ipStr, err)
	}
	var raw []byte
	var layouts [][]int
	if addr.Is4() {
		a := addr.As4()
		raw = a[:]
		layouts = idunnoIPv4Layouts
	} else {
		a := addr.As16()
		raw = a[:]
		layouts = idunnoIPv6Layouts
	}
	return idunnoEncodeBytes(raw, layouts)
}

// idunnoEncodeBytes runs the layout search over the prioritised layout table.
// Returns the first layout's output that produces only valid Unicode codepoints
// (non-overlong, non-surrogate, ≤ U+10FFFF) AND reaches Minimum Confusion Level.
func idunnoEncodeBytes(raw []byte, layouts [][]int) (string, error) {
	totalBits := len(raw) * 8
	bits := bytesToBitstring(raw)

	for _, layout := range layouts {
		sum := 0
		for _, n := range layout {
			sum += n
		}
		// Skip layouts that can't accommodate the address or exceed the
		// §3.1 20-bit padding cap.
		if sum < totalBits || sum-totalBits > idunnoMaxPadBits {
			continue
		}
		cps, ok := tryIDunnoLayout(bits, totalBits, layout)
		if !ok {
			continue
		}
		if !idunnoSatisfiesMinCL(cps) {
			continue
		}
		var sb strings.Builder
		for _, cp := range cps {
			sb.WriteRune(cp)
		}
		return sb.String(), nil
	}
	return "", fmt.Errorf("no RFC 8771 layout reaches Minimum Confusion Level for this address (the universal fallback should handle every real IP — please report this as a bug)")
}

// tryIDunnoLayout attempts to consume `totalBits` bits from `bits` using the
// given chunk-size layout, producing one codepoint per chunk. Padding bits on
// the final chunk are chosen via pickPaddingValue to maximise validity and
// Confusion Level.
func tryIDunnoLayout(bits string, totalBits int, layout []int) ([]rune, bool) {
	cps := make([]rune, 0, len(layout))
	pos := 0
	for i, chunkSize := range layout {
		isLast := i == len(layout)-1
		// How many address bits are available for this chunk?
		availBits := chunkSize
		if pos+chunkSize > totalBits {
			availBits = totalBits - pos
			if !isLast {
				// Only the final chunk may carry padding (RFC §3.1).
				return nil, false
			}
		}
		var value int
		if availBits == chunkSize {
			value = bitsToInt(bits[pos : pos+chunkSize])
			pos += chunkSize
			if !idunnoValidCP(value, chunkSize) {
				return nil, false
			}
		} else {
			padBits := chunkSize - availBits
			addrPart := bits[pos:totalBits]
			pos = totalBits
			chosen, ok := pickPaddingValue(addrPart, padBits, chunkSize)
			if !ok {
				return nil, false
			}
			value = chosen
		}
		cps = append(cps, rune(value))
	}
	return cps, true
}

// pickPaddingValue searches the 2^padBits padding-bit space for a value that,
// concatenated to addrPart, yields a valid codepoint at the chunk's required
// UTF-8 length. Prefers values that are IDNA2008-DISALLOWED (to help reach
// Minimum Confusion Level downstream), then falls back to any valid value.
// Search space is at most 2^20; for typical layouts padBits ≤ 14 so this is
// constant-microsecond fast.
func pickPaddingValue(addrPart string, padBits, chunkSize int) (int, bool) {
	addrVal := 0
	if addrPart != "" {
		addrVal = bitsToInt(addrPart)
	}
	var firstValid int
	foundValid := false
	for p := 0; p < (1 << padBits); p++ {
		v := (addrVal << padBits) | p
		if !idunnoValidCP(v, chunkSize) {
			continue
		}
		if !foundValid {
			firstValid = v
			foundValid = true
		}
		if idunnoDisallowedIDNA2008(rune(v)) {
			return v, true
		}
	}
	if foundValid {
		return firstValid, true
	}
	return 0, false
}

// idunnoValidCP reports whether `value`, viewed as the payload bits of a
// `chunkSize`-bit UTF-8 sequence, forms a valid Unicode codepoint that requires
// EXACTLY that UTF-8 length (no overlong encoding) and is not a surrogate.
func idunnoValidCP(value, chunkSize int) bool {
	if value < 0 || value > 0x10FFFF {
		return false
	}
	switch chunkSize {
	case 7:
		return value < 0x80
	case 11:
		return value >= 0x80 && value <= 0x7FF
	case 16:
		if value < 0x800 || value > 0xFFFF {
			return false
		}
		if value >= 0xD800 && value <= 0xDFFF {
			return false
		}
		return true
	case 21:
		return value >= 0x10000 && value <= 0x10FFFF
	}
	return false
}

// idunnoSatisfiesMinCL reports whether the codepoint sequence reaches the
// Minimum Confusion Level (RFC §4.1): at least one UTF-8 sequence longer than
// one octet AND at least one IDNA2008-DISALLOWED character.
func idunnoSatisfiesMinCL(cps []rune) bool {
	hasMultiOctet := false
	hasDisallowed := false
	for _, cp := range cps {
		if utf8.RuneLen(cp) > 1 {
			hasMultiOctet = true
		}
		if idunnoDisallowedIDNA2008(cp) {
			hasDisallowed = true
		}
	}
	return hasMultiOctet && hasDisallowed
}

// idunnoDisallowedIDNA2008 reports whether the codepoint is DISALLOWED under
// IDNA2008 (RFC 5892) for our purposes. IDNA2008's PVALID set is roughly
// "letters + marks + digits + hyphen". Everything outside that set — symbols,
// punctuation, controls, format characters, private-use, uppercase letters
// that get MAPPED to lowercase — is treated as DISALLOWED here. This is a
// conservative-toward-DISALLOWED classifier: codepoints that IDNA2008 considers
// MAPPED (e.g. ASCII uppercase) are counted as DISALLOWED for §4.1's "at least
// one DISALLOWED character" check, which is faithful to the spirit of the RFC
// (those characters aren't valid in a registered IDN label as-is).
func idunnoDisallowedIDNA2008(cp rune) bool {
	if cp < 0 || cp > 0x10FFFF {
		return true
	}
	// Surrogates: never appear in valid UTF-8; idunnoValidCP rejects them.
	// Belt-and-braces.
	if cp >= 0xD800 && cp <= 0xDFFF {
		return true
	}
	// C0 controls, DEL, C1 controls.
	if cp <= 0x1F || cp == 0x7F || (cp >= 0x80 && cp <= 0x9F) {
		return true
	}
	// ASCII uppercase: MAPPED to lowercase under IDNA2008 → counts as DISALLOWED
	// for our "not directly PVALID" check.
	if cp >= 'A' && cp <= 'Z' {
		return true
	}
	// ASCII PVALID set: lowercase letters, digits, hyphen.
	if cp == '-' {
		return false
	}
	if cp >= '0' && cp <= '9' {
		return false
	}
	if cp >= 'a' && cp <= 'z' {
		return false
	}
	// Non-ASCII: PVALID-ish categories are Letter (Ll, Lm, Lo), Mark (Mn, Mc, Me),
	// and decimal-digit numbers (Nd). Uppercase letters (Lu) and title-case (Lt)
	// are MAPPED → DISALLOWED for us. Everything else (symbols, punctuation,
	// formatting, separators, surrogates, private-use, unassigned) is DISALLOWED.
	if unicode.Is(unicode.Lu, cp) || unicode.Is(unicode.Lt, cp) {
		return true
	}
	if unicode.In(cp, unicode.Ll, unicode.Lm, unicode.Lo, unicode.Mn, unicode.Mc, unicode.Me, unicode.Nd) {
		return false
	}
	return true
}

// IDunnoDecode parses an I-DUNNO string and returns the embedded IP address
// in its canonical text form (dotted-quad for IPv4, lowercase colon-hex for
// IPv6 per RFC 5952). Total bit-payload disambiguates IPv4 (32 address bits
// + 0–20 pad → 32–52 codepoint bits) from IPv6 (128 + 0–20 → 128–148 bits);
// those ranges don't overlap.
//
// §3.2 of RFC 8771 says "this section is intentionally omitted. The machines
// will know how to do it." This is the machines knowing how to do it.
func IDunnoDecode(s string) (string, error) {
	if s == "" {
		return "", fmt.Errorf("input must not be empty")
	}
	if !utf8.ValidString(s) {
		return "", fmt.Errorf("input is not valid UTF-8")
	}
	var bits strings.Builder
	for _, cp := range s {
		bitsPerCP, err := idunnoBitsForCodepoint(cp)
		if err != nil {
			return "", err
		}
		for i := bitsPerCP - 1; i >= 0; i-- {
			if (cp>>i)&1 == 1 {
				bits.WriteByte('1')
			} else {
				bits.WriteByte('0')
			}
		}
	}
	total := bits.Len()
	var addrBits int
	switch {
	case total >= 32 && total <= 32+idunnoMaxPadBits:
		addrBits = 32
	case total >= 128 && total <= 128+idunnoMaxPadBits:
		addrBits = 128
	default:
		return "", fmt.Errorf("I-DUNNO total bit-payload %d does not match IPv4 [32-52] or IPv6 [128-148]", total)
	}
	addrField := bits.String()[:addrBits]
	raw := bitstringToBytes(addrField)
	var addr netip.Addr
	if addrBits == 32 {
		var a4 [4]byte
		copy(a4[:], raw)
		addr = netip.AddrFrom4(a4)
	} else {
		var a16 [16]byte
		copy(a16[:], raw)
		addr = netip.AddrFrom16(a16).Unmap()
	}
	return addr.String(), nil
}

// idunnoBitsForCodepoint maps a Unicode codepoint to the number of bits its
// UTF-8 encoding carries per RFC 8771 §3 Table 1.
func idunnoBitsForCodepoint(cp rune) (int, error) {
	switch {
	case cp < 0:
		return 0, fmt.Errorf("invalid codepoint U+%X (negative)", cp)
	case cp < 0x80:
		return 7, nil
	case cp < 0x800:
		return 11, nil
	case cp >= 0xD800 && cp <= 0xDFFF:
		return 0, fmt.Errorf("invalid codepoint U+%04X (surrogate range)", cp)
	case cp < 0x10000:
		return 16, nil
	case cp <= 0x10FFFF:
		return 21, nil
	default:
		return 0, fmt.Errorf("invalid codepoint U+%X (> U+10FFFF)", cp)
	}
}

// bytesToBitstring renders raw bytes as a string of '0' and '1' characters,
// MSB first within each byte, byte-by-byte.
func bytesToBitstring(raw []byte) string {
	var sb strings.Builder
	sb.Grow(len(raw) * 8)
	for _, b := range raw {
		for i := 7; i >= 0; i-- {
			if (b>>i)&1 == 1 {
				sb.WriteByte('1')
			} else {
				sb.WriteByte('0')
			}
		}
	}
	return sb.String()
}

// bitstringToBytes converts a '0'/'1' string back to bytes. Length must be
// a multiple of 8 (always true here since we only call it with 32 or 128).
func bitstringToBytes(bits string) []byte {
	out := make([]byte, len(bits)/8)
	for i := range out {
		var b byte
		for j := 0; j < 8; j++ {
			if bits[i*8+j] == '1' {
				b |= 1 << (7 - j)
			}
		}
		out[i] = b
	}
	return out
}

// bitsToInt parses a '0'/'1' string into an int (≤ 32 bits expected).
func bitsToInt(s string) int {
	v := 0
	for _, c := range s {
		v <<= 1
		if c == '1' {
			v |= 1
		}
	}
	return v
}
