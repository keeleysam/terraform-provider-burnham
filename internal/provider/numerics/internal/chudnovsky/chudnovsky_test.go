package chudnovsky

import (
	"strings"
	"testing"
)

// piFirst100Reference is the canonical first 100 decimal digits of π *following* the decimal point (the leading 3 is stripped). Sourced from multiple independent references (Wikipedia "Pi", OEIS A000796, et al.) and hard-coded here so a corrupted Chudnovsky implementation fails this test.
const piFirst100Reference = "1415926535897932384626433832795028841971693993751058209749445923078164062862089986280348253421170679"

func TestPiDigits_First100(t *testing.T) {
	got := PiDigits(100)
	if got != piFirst100Reference {
		t.Fatalf("Chudnovsky disagrees with reference π in the first 100 digits.\n got: %s\nwant: %s", got, piFirst100Reference)
	}
}

func TestPiDigits_LengthMatchesRequest(t *testing.T) {
	for _, n := range []int{1, 10, 50, 100, 500, 1000, 5000} {
		got := PiDigits(n)
		if len(got) != n {
			t.Errorf("PiDigits(%d) returned %d chars, want %d", n, len(got), n)
		}
		// Every character must be an ASCII digit.
		for i, r := range got {
			if r < '0' || r > '9' {
				t.Errorf("PiDigits(%d)[%d] = %q (not a digit)", n, i, r)
				break
			}
		}
	}
}

func TestPiDigits_PrefixesAreConsistent(t *testing.T) {
	// Computing N digits should agree with the first M digits computed at a larger precision, for M < N. This guards against precision-bleed bugs where the lower-order digits of a higher-precision call can perturb the upper-order digits.
	thousand := PiDigits(1000)
	for _, m := range []int{10, 50, 100, 500} {
		got := PiDigits(m)
		if got != thousand[:m] {
			t.Errorf("PiDigits(%d) = %s; thousand[:%d] = %s", m, got, m, thousand[:m])
		}
	}
}

func TestPiDigits_LeadingThreeIsStripped(t *testing.T) {
	// The first character of the result must be the first digit *after* the decimal point — '1' (since π = 3.1415…). If we accidentally include the integer part, we'd see '3' here.
	got := PiDigits(1)
	if got != "1" {
		t.Fatalf("PiDigits(1) = %q, want \"1\" (per RFC 3091's implied leading 3)", got)
	}
}

func TestPiDigits_FullReferenceMatches(t *testing.T) {
	// Verify the first 100 digits also appear at the start of a longer
	// computation — protects against off-by-one in the slicing.
	long := PiDigits(200)
	if !strings.HasPrefix(long, piFirst100Reference) {
		t.Fatalf("first 100 digits of PiDigits(200) do not match reference\n got: %s\nwant prefix: %s", long[:100], piFirst100Reference)
	}
}
