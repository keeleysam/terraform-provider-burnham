package dpd

import "testing"

// ieeeTestVectors are the encoding examples from Wikipedia's "Densely packed decimal" article (which mirrors the IEEE 754-2008 spec). Used as the primary correctness witness.
var ieeeTestVectors = []struct {
	digits [3]byte
	dpd    uint16
}{
	{[3]byte{0, 0, 5}, 0b0000000101}, // "005" — all small
	{[3]byte{0, 5, 5}, 0b0001010101}, // "055" — all small
	{[3]byte{0, 7, 9}, 0b0001111001}, // "079" — c large
	{[3]byte{0, 8, 0}, 0b0000001010}, // "080" — b large
	{[3]byte{0, 9, 9}, 0b0001011111}, // "099" — b,c large
	{[3]byte{5, 5, 5}, 0b1011010101}, // "555" — all small
	{[3]byte{8, 0, 0}, 0b0000001100}, // "800" — a large
	{[3]byte{8, 8, 0}, 0b0000001110}, // "880" — a,b large
	{[3]byte{8, 0, 8}, 0b0000101110}, // "808" — a,c large
	{[3]byte{8, 8, 8}, 0b0001101110}, // "888" — all large (canonical: high two bits = 0)
	{[3]byte{9, 9, 9}, 0b0011111111}, // "999" — all large
}

// Note on "888" / "999" / etc.: the all-large case has two don't-care bits in the IEEE encoding, so each all-large digit triple has 4 valid DPD aliases. Encode emits the canonical form with the don't-care bits set to 0; Decode accepts any of the aliases (verified separately by TestDecodeHandlesRedundantEncodings).

func TestEncode_IEEEVectors(t *testing.T) {
	for _, tc := range ieeeTestVectors {
		got := Encode(tc.digits[0], tc.digits[1], tc.digits[2])
		if got != tc.dpd {
			t.Errorf("Encode(%d, %d, %d) = %010b, want %010b",
				tc.digits[0], tc.digits[1], tc.digits[2], got, tc.dpd)
		}
	}
}

func TestDecode_IEEEVectors(t *testing.T) {
	for _, tc := range ieeeTestVectors {
		d0, d1, d2 := Decode(tc.dpd)
		if d0 != tc.digits[0] || d1 != tc.digits[1] || d2 != tc.digits[2] {
			t.Errorf("Decode(%010b) = (%d,%d,%d), want (%d,%d,%d)",
				tc.dpd, d0, d1, d2, tc.digits[0], tc.digits[1], tc.digits[2])
		}
	}
}

// TestRoundtrip exhaustively encodes every (d0, d1, d2) triple in [0,9]³ (1000 cases), decodes the result, and asserts equality. Catches any asymmetry between the encoder and decoder.
func TestRoundtrip(t *testing.T) {
	for d0 := byte(0); d0 < 10; d0++ {
		for d1 := byte(0); d1 < 10; d1++ {
			for d2 := byte(0); d2 < 10; d2++ {
				code := Encode(d0, d1, d2)
				got0, got1, got2 := Decode(code)
				if got0 != d0 || got1 != d1 || got2 != d2 {
					t.Errorf("roundtrip (%d,%d,%d) → 0x%03x → (%d,%d,%d)",
						d0, d1, d2, code, got0, got1, got2)
				}
			}
		}
	}
}

// TestEncodeProducesValidDPDRange asserts that all 1000 valid 3-digit inputs encode to values in [0, 1023] (10 bits). Defends against accidentally setting bits outside the low 10.
func TestEncodeProducesValidDPDRange(t *testing.T) {
	for d0 := byte(0); d0 < 10; d0++ {
		for d1 := byte(0); d1 < 10; d1++ {
			for d2 := byte(0); d2 < 10; d2++ {
				code := Encode(d0, d1, d2)
				if code >= 1024 {
					t.Errorf("Encode(%d,%d,%d) = %d, exceeds 10 bits", d0, d1, d2, code)
				}
			}
		}
	}
}

// TestDecodeHandlesRedundantEncodings: per IEEE 754-2008, several 10-bit patterns decode to digit triples that have a different canonical encoding (because the all-large case has 2 don't-care bits → 4 redundant aliases per all-large triple). Decode should still return the correct digit triple; only Encode is canonical.
func TestDecodeHandlesRedundantEncodings(t *testing.T) {
	// All four aliases of "999" — DPD's all-large case has two don't-care
	// bits (p9, p8), giving 2² = 4 valid encodings of the same digit triple.
	for hi := uint16(0); hi < 4; hi++ {
		// p9 p8 free; p3 p2 p1 = 1 1 1; p7 = c = 1; p4 = f = 1; p0 = i = 1; p6 p5 = 1 1.
		code := (hi << 8) | 0b00_1111_1111
		d0, d1, d2 := Decode(code)
		if d0 != 9 || d1 != 9 || d2 != 9 {
			t.Errorf("Decode redundant 999 alias %010b = (%d,%d,%d), want (9,9,9)", code, d0, d1, d2)
		}
	}
}
