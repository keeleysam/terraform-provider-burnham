/*
Package dpd implements Densely Packed Decimal encoding and decoding per
IEEE 754-2008 §3.5.2.

DPD packs three decimal digits (each 0-9) into a 10-bit code, achieving
3.33 bits per digit — within 0.3% of the information-theoretic limit
(log₂10 ≈ 3.322 bits/digit). It's the encoding used by the IEEE
decimal32 / decimal64 / decimal128 floating-point formats.

The encoding is a specific bit permutation chosen so a small lookup table
(or a handful of boolean operations) round-trips between BCD and DPD.
For our use — packing a fixed table of digits of π — we use the boolean-
equation form: no tables, no init, just a switch over the 8 cases of
"which of the three digits are large (≥ 8)".

References:
  - IEEE 754-2008 (and 754-2019) §3.5.2 "Decimal interchange format
    encodings"
  - Mike Cowlishaw, "Densely Packed Decimal Encoding"
    http://speleotrove.com/decimal/DPDecimal.html
  - Wikipedia: https://en.wikipedia.org/wiki/Densely_packed_decimal

Naming convention used below follows Wikipedia's table:

  Three BCD digits as four-bit groups d0, d1, d2 (d0 is the most-significant
  digit, d2 the least). Each digit's bits are written (msb → lsb) as

      d0 = a3 a2 a1 a0     where the lower three bits {a2, a1, a0} are
      d1 = b3 b2 b1 b0     called {a, b, c}, {d, e, f}, {g, h, i}
      d2 = c3 c2 c1 c0     respectively, and a3, b3, c3 are the high bits.

  The 10-bit DPD output is named bit-by-bit as p9 p8 p7 p6 p5 p4 p3 p2 p1 p0.

  The low bit of each digit (a0, b0, c0) always passes through unchanged —
  to DPD positions p7, p4, p0.

The 8 encoding cases dispatched on (a3, b3, c3):

  (0,0,0)  all small         p9..p0 = a b c d e f 0 g h i
  (0,0,1)  c large            p9..p0 = a b c d e f 1 0 0 i
  (0,1,0)  b large            p9..p0 = a b c g h f 1 0 1 i
  (1,0,0)  a large            p9..p0 = g h c d e f 1 1 0 i
  (1,1,0)  a, b large         p9..p0 = g h c 0 0 f 1 1 1 i
  (1,0,1)  a, c large         p9..p0 = d e c 0 1 f 1 1 1 i
  (0,1,1)  b, c large         p9..p0 = a b c 1 0 f 1 1 1 i
  (1,1,1)  all large          p9..p0 = . . c 1 1 f 1 1 1 i  (we emit 0 for "."  — those are don't-cares)

Decoding inverts the encoding by examining p3 (and, when p3=1, p2 p1 to
distinguish the four "all big" / "two big" cases; when p3 p2 p1 = 1 1 1,
also p6 p5 to pick among the four "many large" sub-cases).

Test vectors (verified against Wikipedia):

  005 (digits 0,0,5) → 00 0000 0101
  055 (digits 0,5,5) → 00 0101 0101
  080 (digits 0,8,0) → 00 0000 1010
  555 (digits 5,5,5) → 10 1101 0101
  999 (digits 9,9,9) → 00 1111 1111
*/

package dpd

// Encode packs three BCD digits into a 10-bit DPD code per IEEE 754-2008.
// The high digit is d0, middle d1, low d2; each must be in [0, 9].
// The result fits in the low 10 bits of the returned uint16.
func Encode(d0, d1, d2 byte) uint16 {
	a3 := (d0 >> 3) & 1
	a := (d0 >> 2) & 1
	b := (d0 >> 1) & 1
	c := d0 & 1

	b3 := (d1 >> 3) & 1
	d := (d1 >> 2) & 1
	e := (d1 >> 1) & 1
	f := d1 & 1

	c3 := (d2 >> 3) & 1
	g := (d2 >> 2) & 1
	h := (d2 >> 1) & 1
	i := d2 & 1

	// Pack p9..p0 directly per the case table. Each variable below
	// becomes one of the 10 output bits; we OR them together at the end.
	// Naming: pN holds the (already-shifted) value of bit pN.
	switch (a3 << 2) | (b3 << 1) | c3 {

	case 0b000: // all small: a b c d e f 0 g h i
		return u16(a, 9) | u16(b, 8) | u16(c, 7) |
			u16(d, 6) | u16(e, 5) | u16(f, 4) |
			u16(0, 3) |
			u16(g, 2) | u16(h, 1) | u16(i, 0)

	case 0b001: // c large: a b c d e f 1 0 0 i
		return u16(a, 9) | u16(b, 8) | u16(c, 7) |
			u16(d, 6) | u16(e, 5) | u16(f, 4) |
			u16(1, 3) | u16(0, 2) | u16(0, 1) | u16(i, 0)

	case 0b010: // b large: a b c g h f 1 0 1 i
		return u16(a, 9) | u16(b, 8) | u16(c, 7) |
			u16(g, 6) | u16(h, 5) | u16(f, 4) |
			u16(1, 3) | u16(0, 2) | u16(1, 1) | u16(i, 0)

	case 0b100: // a large: g h c d e f 1 1 0 i
		return u16(g, 9) | u16(h, 8) | u16(c, 7) |
			u16(d, 6) | u16(e, 5) | u16(f, 4) |
			u16(1, 3) | u16(1, 2) | u16(0, 1) | u16(i, 0)

	case 0b110: // a, b large: g h c 0 0 f 1 1 1 i
		return u16(g, 9) | u16(h, 8) | u16(c, 7) |
			u16(0, 6) | u16(0, 5) | u16(f, 4) |
			u16(1, 3) | u16(1, 2) | u16(1, 1) | u16(i, 0)

	case 0b101: // a, c large: d e c 0 1 f 1 1 1 i
		return u16(d, 9) | u16(e, 8) | u16(c, 7) |
			u16(0, 6) | u16(1, 5) | u16(f, 4) |
			u16(1, 3) | u16(1, 2) | u16(1, 1) | u16(i, 0)

	case 0b011: // b, c large: a b c 1 0 f 1 1 1 i
		return u16(a, 9) | u16(b, 8) | u16(c, 7) |
			u16(1, 6) | u16(0, 5) | u16(f, 4) |
			u16(1, 3) | u16(1, 2) | u16(1, 1) | u16(i, 0)

	case 0b111: // all large: 0 0 c 1 1 f 1 1 1 i (high two bits don't-care; we emit 0)
		return u16(0, 9) | u16(0, 8) | u16(c, 7) |
			u16(1, 6) | u16(1, 5) | u16(f, 4) |
			u16(1, 3) | u16(1, 2) | u16(1, 1) | u16(i, 0)
	}
	// unreachable; the switch covers all 8 values of a 3-bit input
	return 0
}

// Decode reverses Encode. Only the low 10 bits of dpd are used; high bits are
// ignored. Returns three digits each guaranteed to be in [0, 9].
func Decode(dpd uint16) (d0, d1, d2 byte) {
	// Extract the relevant DPD bits.
	p9 := byte((dpd >> 9) & 1)
	p8 := byte((dpd >> 8) & 1)
	p7 := byte((dpd >> 7) & 1)
	p6 := byte((dpd >> 6) & 1)
	p5 := byte((dpd >> 5) & 1)
	p4 := byte((dpd >> 4) & 1)
	p3 := byte((dpd >> 3) & 1)
	p2 := byte((dpd >> 2) & 1)
	p1 := byte((dpd >> 1) & 1)
	p0 := byte(dpd & 1)

	// Mnemonics matching the encoding table.
	c := p7 // d0's low bit, always.
	f := p4 // d1's low bit, always.
	i := p0 // d2's low bit, always.

	if p3 == 0 {
		// All small: d0 = 0 a b c, d1 = 0 d e f, d2 = 0 g h i
		// where a b c = p9 p8 p7, d e f = p6 p5 p4, g h i = p2 p1 p0.
		d0 = (p9 << 2) | (p8 << 1) | c
		d1 = (p6 << 2) | (p5 << 1) | f
		d2 = (p2 << 2) | (p1 << 1) | i
		return
	}

	// p3 == 1.
	switch (p2 << 1) | p1 {
	case 0b00: // c only large: d2 = 1 0 0 i; d0,d1 small from p9 p8 p7 / p6 p5 p4
		d0 = (p9 << 2) | (p8 << 1) | c
		d1 = (p6 << 2) | (p5 << 1) | f
		d2 = 0b1000 | i
		return

	case 0b01: // b only large: d1 = 1 0 0 f; d0 small (a b c at p9 p8 p7); d2 small (g h i at p6 p5 p0)
		d0 = (p9 << 2) | (p8 << 1) | c
		d1 = 0b1000 | f
		d2 = (p6 << 2) | (p5 << 1) | i
		return

	case 0b10: // a only large: d0 = 1 0 0 c; d1 small (d e f at p6 p5 p4); d2 small (g h i at p9 p8 p0)
		d0 = 0b1000 | c
		d1 = (p6 << 2) | (p5 << 1) | f
		d2 = (p9 << 2) | (p8 << 1) | i
		return

	case 0b11: // p3 p2 p1 = 1 1 1 — sub-dispatch on p6 p5
		switch (p6 << 1) | p5 {
		case 0b00: // a,b large: d0 = 1 0 0 c; d1 = 1 0 0 f; d2 small (g h i at p9 p8 p0)
			d0 = 0b1000 | c
			d1 = 0b1000 | f
			d2 = (p9 << 2) | (p8 << 1) | i
			return
		case 0b01: // a,c large: d0 = 1 0 0 c; d1 small (d e f at p9 p8 p4); d2 = 1 0 0 i
			d0 = 0b1000 | c
			d1 = (p9 << 2) | (p8 << 1) | f
			d2 = 0b1000 | i
			return
		case 0b10: // b,c large: d0 small (a b c at p9 p8 p7); d1 = 1 0 0 f; d2 = 1 0 0 i
			d0 = (p9 << 2) | (p8 << 1) | c
			d1 = 0b1000 | f
			d2 = 0b1000 | i
			return
		case 0b11: // all large
			d0 = 0b1000 | c
			d1 = 0b1000 | f
			d2 = 0b1000 | i
			return
		}
	}
	return // unreachable
}

// u16 places `bit` (0 or 1) at position `pos` in a uint16.
func u16(bit byte, pos uint) uint16 {
	return uint16(bit) << pos
}
