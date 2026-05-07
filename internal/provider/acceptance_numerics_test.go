package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

// ─── pi_digit (RFC 3091 §2.1.2 UDP reply for π) ────────────────────────

func TestAcc_PiDigit_FirstDigit(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::pi_digit(1) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("1:1")),
	)
}

func TestAcc_PiDigit_HundredthDigit(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::pi_digit(100) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("100:9")),
	)
}

func TestAcc_PiDigit_FeynmanPoint(t *testing.T) {
	// digit 762 is the start of "999999".
	runOutputTest(t,
		`output "test" { value = provider::burnham::pi_digit(762) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("762:9")),
	)
}

func TestAcc_PiDigit_MillionthDigit(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::pi_digit(1000000) }`,
		// digit 1,000,000 of π is '1' (verified against multiple references)
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("1000000:1")),
	)
}

func TestAcc_PiDigit_AtCap(t *testing.T) {
	// At our embedded cap (3,141,592 = floor(π × 10^6)), digit value is '4'.
	// Verified at table-generation time against the Chudnovsky-produced packed
	// table; the value is locked in pi_packed.bin.
	runOutputTest(t,
		`output "test" { value = provider::burnham::pi_digit(3141592) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("3141592:4")),
	)
}

func TestAcc_PiDigit_RejectsZero(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::pi_digit(0) }`,
		regexp.MustCompile(`(?is)n\s+>=\s+1`),
	)
}

func TestAcc_PiDigit_BeyondCap(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::pi_digit(3141593) }`,
		regexp.MustCompile(`(?is)supports\s+(n|count)\s+up\s+to`),
	)
}

// ─── pi_digits (RFC 3091 §1 TCP service for π) ─────────────────────────

func TestAcc_PiDigits_FirstTen(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::pi_digits(10) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("1415926535")),
	)
}

func TestAcc_PiDigits_Empty(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::pi_digits(0) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("")),
	)
}

func TestAcc_PiDigits_RejectsNegative(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::pi_digits(-1) }`,
		regexp.MustCompile(`(?is)count\s+must\s+be\s+>=\s+0`),
	)
}

func TestAcc_PiDigits_BeyondCap(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::pi_digits(3141593) }`,
		regexp.MustCompile(`(?is)supports\s+(n|count)\s+up\s+to`),
	)
}

// ─── pi_approximate_digit (RFC 3091 §2.2 UDP reply for 22/7) ───────────

func TestAcc_PiApproximateDigit_OneCycle(t *testing.T) {
	cases := []struct {
		n    int
		want string
	}{
		{1, "1:1"},
		{2, "2:4"},
		{3, "3:2"},
		{4, "4:8"},
		{5, "5:5"},
		{6, "6:7"},
	}
	for _, c := range cases {
		t.Run(c.want, func(t *testing.T) {
			runOutputTest(t,
				"output \"test\" { value = provider::burnham::pi_approximate_digit("+itoa(c.n)+") }",
				statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact(c.want)),
			)
		})
	}
}

func TestAcc_PiApproximateDigit_CycleWrap(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::pi_approximate_digit(7) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("7:1")),
	)
}

func TestAcc_PiApproximateDigit_MillionthDigit(t *testing.T) {
	// (1_000_000 - 1) mod 6 = 999_999 mod 6 = 3 → "142857"[3] = '8'
	runOutputTest(t,
		`output "test" { value = provider::burnham::pi_approximate_digit(1000000) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("1000000:8")),
	)
}

func TestAcc_PiApproximateDigit_RejectsZero(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::pi_approximate_digit(0) }`,
		regexp.MustCompile(`(?is)n\s+>=\s+1`),
	)
}

func TestAcc_PiApproximateDigit_RejectsFractional(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::pi_approximate_digit(1.5) }`,
		regexp.MustCompile(`(?is)whole\s+number`),
	)
}

// ─── pi_approximate_digits (RFC 3091 §1.1 TCP for 22/7) ────────────────

func TestAcc_PiApproximateDigits_TwoCycles(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::pi_approximate_digits(12) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("142857142857")),
	)
}

func TestAcc_PiApproximateDigits_Partial(t *testing.T) {
	// 5 digits should give exactly the first 5 chars of the cycle.
	runOutputTest(t,
		`output "test" { value = provider::burnham::pi_approximate_digits(5) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("14285")),
	)
}

func TestAcc_PiApproximateDigits_Empty(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::pi_approximate_digits(0) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("")),
	)
}

func TestAcc_PiApproximateDigits_RejectsNegative(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::pi_approximate_digits(-1) }`,
		regexp.MustCompile(`(?is)count\s+must\s+be\s+>=\s+0`),
	)
}

// itoa is a tiny helper local to this file to avoid importing strconv just to interpolate a small int.
func itoa(n int) string {
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
