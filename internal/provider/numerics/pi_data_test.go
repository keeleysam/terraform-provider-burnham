package numerics

import (
	"strings"
	"testing"

	"github.com/keeleysam/terraform-burnham/internal/provider/numerics/internal/dpd"
)

// piFirst100Reference is the canonical first 100 decimal digits of π *following* the decimal point. Sourced from multiple independent references (Wikipedia "Pi", OEIS A000796) and hard-coded so a corrupted pi_packed.bin fails this test.
const piFirst100Reference = "1415926535897932384626433832795028841971693993751058209749445923078164062862089986280348253421170679"

func TestPiPacked_FileSize(t *testing.T) {
	// DPD packs three digits in ten bits. ⌈piEmbeddedDigitCount/3⌉ triples;
	// each triple is 10 bits; the byte count is ⌈totalBits / 8⌉.
	triples := (piEmbeddedDigitCount + 2) / 3
	wantBytes := (triples*10 + 7) / 8
	if got := len(piPackedBytes); got != wantBytes {
		t.Fatalf("piPackedBytes length = %d, want %d (DPD: %d triples × 10 bits)", got, wantBytes, triples)
	}
}

func TestPiPacked_AllTriplesDecodeToValidDigits(t *testing.T) {
	// Iterate every full triple in the packed table and assert each decodes
	// to digits in [0, 9]. A nibble out of range would mean dpd.Decode has
	// a bug; in practice this also catches "the file got truncated and we're
	// reading past valid data into garbage". Partial last triple is skipped
	// — its trailing positions are encoder-padding zeros, not real digits.
	fullTriples := int64(piEmbeddedDigitCount / 3)
	for t0 := int64(0); t0 < fullTriples; t0++ {
		d0, d1, d2 := dpd.Decode(readDPDTriple(t0))
		if d0 > 9 || d1 > 9 || d2 > 9 {
			t.Fatalf("triple %d decoded to (%d, %d, %d); all must be in [0,9]", t0, d0, d1, d2)
		}
	}
}

func TestPiDigitChar_FirstHundredMatchesReference(t *testing.T) {
	for i := int64(1); i <= 100; i++ {
		got := piDigitChar(i)
		want := piFirst100Reference[i-1]
		if got != want {
			t.Errorf("piDigitChar(%d) = %q, want %q", i, got, want)
		}
	}
}

func TestPiDigitChar_KnownPositions(t *testing.T) {
	// Spot-checks at positions documented in widely cited sources. These
	// catch off-by-one errors in piDigitChar's nibble extraction.
	cases := []struct {
		n    int64
		want byte
		why  string
	}{
		{1, '1', "3.[1]415…"},
		{2, '4', "3.1[4]15…"},
		{3, '1', "3.14[1]5…"},
		{10, '5', "first 10 digits are 1415926535"},
		{100, '9', "first 100 digits end in '79' — digit 100 is '9'"},
		// The Feynman point: digits 762..767 of π are "999999". This is the
		// most famous long run of repeating digits and a classic sanity check.
		// See e.g. https://en.wikipedia.org/wiki/Six_nines_in_pi.
		{762, '9', "Feynman point digit 1"},
		{763, '9', "Feynman point digit 2"},
		{764, '9', "Feynman point digit 3"},
		{765, '9', "Feynman point digit 4"},
		{766, '9', "Feynman point digit 5"},
		{767, '9', "Feynman point digit 6"},
		{768, '8', "digit immediately after Feynman point is '8'"},
	}
	for _, c := range cases {
		got := piDigitChar(c.n)
		if got != c.want {
			t.Errorf("piDigitChar(%d) = %q, want %q  // %s", c.n, got, c.want, c.why)
		}
	}
}

func TestPiFirstNDigits_Lengths(t *testing.T) {
	for _, n := range []int64{0, 1, 10, 100, 1000, 100_000, 1_000_000, piEmbeddedDigitCount} {
		got := piFirstNDigits(n)
		if int64(len(got)) != n {
			t.Errorf("piFirstNDigits(%d) returned %d chars, want %d", n, len(got), n)
		}
	}
}

func TestPiFirstNDigits_AllAsciiDigits(t *testing.T) {
	s := piFirstNDigits(piEmbeddedDigitCount)
	for i, c := range []byte(s) {
		if c < '0' || c > '9' {
			t.Fatalf("piFirstNDigits(%d)[%d] = %q (not a digit)", piEmbeddedDigitCount, i, c)
		}
	}
}

func TestPiFirstNDigits_FirstHundredMatchesReference(t *testing.T) {
	got := piFirstNDigits(100)
	if got != piFirst100Reference {
		t.Fatalf("piFirstNDigits(100) = %s\nwant: %s", got, piFirst100Reference)
	}
}

func TestPiFirstNDigits_PrefixOfLonger(t *testing.T) {
	// The first 100 digits should be a prefix of any longer slice.
	long := piFirstNDigits(piEmbeddedDigitCount)
	if !strings.HasPrefix(long, piFirst100Reference) {
		t.Fatal("first 100 digits of full embed do not match reference")
	}
}
