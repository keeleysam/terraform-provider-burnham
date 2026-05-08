package provider

import (
	"math/big"
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

func TestAcc_PiApproximateDigits_RejectsAboveCap(t *testing.T) {
	// Regression: previously `pi_approximate_digits(MaxInt64)` would `make([]byte, MaxInt)` and OOM. Now capped at piApproximateMaxDigits (= ⌊π × 10⁶⌋ = 3,141,592) — same as `pi_digits`.
	runErrorTest(t,
		`output "test" { value = provider::burnham::pi_approximate_digits(3141593) }`,
		regexp.MustCompile(`(?is)count\s+must\s+be\s+<=\s+3141592`),
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

// bigF builds a *big.Float for NumberExact assertions. Panics on a malformed literal because that's a programmer error in the test, not a runtime concern.
func bigF(s string) *big.Float {
	v, _, err := big.ParseFloat(s, 10, 256, big.ToNearestEven)
	if err != nil {
		panic("bigF: bad literal " + s + ": " + err.Error())
	}
	return v
}

// ─── mean ──────────────────────────────────────────────────────────────

func TestAcc_Mean_SimpleAverage(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::mean([1, 2, 3, 4, 5]) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NumberExact(bigF("3"))),
	)
}

func TestAcc_Mean_FractionalResult(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::mean([1, 2]) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NumberExact(bigF("1.5"))),
	)
}

func TestAcc_Mean_Single(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::mean([42]) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NumberExact(bigF("42"))),
	)
}

func TestAcc_Mean_RejectsEmpty(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::mean([]) }`,
		regexp.MustCompile(`(?is)at\s+least\s+one\s+value`),
	)
}

// ─── median ────────────────────────────────────────────────────────────

func TestAcc_Median_OddCount(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::median([5, 1, 3, 2, 4]) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NumberExact(bigF("3"))),
	)
}

func TestAcc_Median_EvenCount(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::median([1, 2, 3, 4]) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NumberExact(bigF("2.5"))),
	)
}

func TestAcc_Median_Single(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::median([7]) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NumberExact(bigF("7"))),
	)
}

func TestAcc_Median_RejectsEmpty(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::median([]) }`,
		regexp.MustCompile(`(?is)at\s+least\s+one\s+value`),
	)
}

// ─── percentile ────────────────────────────────────────────────────────

func TestAcc_Percentile_Zero(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::percentile([10, 20, 30, 40], 0) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NumberExact(bigF("10"))),
	)
}

func TestAcc_Percentile_Hundred(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::percentile([10, 20, 30, 40], 100) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NumberExact(bigF("40"))),
	)
}

func TestAcc_Percentile_50MatchesMedian(t *testing.T) {
	// numpy: percentile([1,2,3,4,5], 50) == 3
	runOutputTest(t,
		`output "test" { value = provider::burnham::percentile([1, 2, 3, 4, 5], 50) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NumberExact(bigF("3"))),
	)
}

func TestAcc_Percentile_LinearInterpolation(t *testing.T) {
	// numpy: percentile([1, 2, 3, 4], 75) == 3.25
	// h = 0.75 * 3 = 2.25; floor = 2; frac = 0.25; sorted[2]=3, sorted[3]=4; 3 + 0.25*1 = 3.25
	runOutputTest(t,
		`output "test" { value = provider::burnham::percentile([1, 2, 3, 4], 75) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NumberExact(bigF("3.25"))),
	)
}

func TestAcc_Percentile_OutOfRange(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::percentile([1, 2, 3], 150) }`,
		regexp.MustCompile(`(?is)p\s+must\s+be\s+in\s+\[0,\s*100\]`),
	)
}

func TestAcc_Percentile_RejectsNegative(t *testing.T) {
	// Lock the lower bound — `p < 0` must error symmetrically with `p > 100`.
	runErrorTest(t,
		`output "test" { value = provider::burnham::percentile([1, 2, 3], -1) }`,
		regexp.MustCompile(`(?is)p\s+must\s+be\s+in\s+\[0,\s*100\]`),
	)
}

func TestAcc_Percentile_RejectsEmpty(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::percentile([], 50) }`,
		regexp.MustCompile(`(?is)at\s+least\s+one\s+value`),
	)
}

// ─── variance / stddev ─────────────────────────────────────────────────

func TestAcc_Variance_PopulationFormula(t *testing.T) {
	// Population variance of [2, 4, 4, 4, 5, 5, 7, 9]: mean = 5; squared deviations sum = 32; var = 32/8 = 4
	runOutputTest(t,
		`output "test" { value = provider::burnham::variance([2, 4, 4, 4, 5, 5, 7, 9]) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NumberExact(bigF("4"))),
	)
}

func TestAcc_Variance_SingleElementIsZero(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::variance([99]) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NumberExact(bigF("0"))),
	)
}

func TestAcc_Variance_RejectsEmpty(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::variance([]) }`,
		regexp.MustCompile(`(?is)at\s+least\s+one\s+value`),
	)
}

func TestAcc_Stddev_PopulationFormula(t *testing.T) {
	// stddev of the same set above is sqrt(4) = 2
	runOutputTest(t,
		`output "test" { value = provider::burnham::stddev([2, 4, 4, 4, 5, 5, 7, 9]) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NumberExact(bigF("2"))),
	)
}

func TestAcc_Stddev_SingleElementIsZero(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::stddev([99]) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NumberExact(bigF("0"))),
	)
}

// ─── mode ──────────────────────────────────────────────────────────────

func TestAcc_Mode_Unimodal(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::mode([1, 2, 2, 3]) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ListExact([]knownvalue.Check{
			knownvalue.NumberExact(bigF("2")),
		})),
	)
}

func TestAcc_Mode_Bimodal(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::mode([1, 1, 2, 2, 3]) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ListExact([]knownvalue.Check{
			knownvalue.NumberExact(bigF("1")),
			knownvalue.NumberExact(bigF("2")),
		})),
	)
}

func TestAcc_Mode_AllUnique(t *testing.T) {
	// All values appear once → no mode → empty list. (Echoing the input would mislead callers using `mode` to detect repetition.)
	runOutputTest(t,
		`output "test" { value = provider::burnham::mode([3, 1, 2]) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ListExact([]knownvalue.Check{})),
	)
}

func TestAcc_Mode_SingleElement(t *testing.T) {
	// Degenerate one-element case: the value is trivially "the mode".
	runOutputTest(t,
		`output "test" { value = provider::burnham::mode([5]) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ListExact([]knownvalue.Check{
			knownvalue.NumberExact(bigF("5")),
		})),
	)
}

func TestAcc_Mode_RejectsEmpty(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::mode([]) }`,
		regexp.MustCompile(`(?is)at\s+least\s+one\s+value`),
	)
}

// ─── mod_floor ─────────────────────────────────────────────────────────

func TestAcc_ModFloor_PositivePositive(t *testing.T) {
	// 7 mod 3 = 1
	runOutputTest(t,
		`output "test" { value = provider::burnham::mod_floor(7, 3) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NumberExact(bigF("1"))),
	)
}

func TestAcc_ModFloor_NegativePositive(t *testing.T) {
	// -7 mod 3 = 2 (sign of divisor; the whole point of this function vs %)
	runOutputTest(t,
		`output "test" { value = provider::burnham::mod_floor(-7, 3) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NumberExact(bigF("2"))),
	)
}

func TestAcc_ModFloor_PositiveNegative(t *testing.T) {
	// 7 mod -3 = -2 (sign of divisor)
	runOutputTest(t,
		`output "test" { value = provider::burnham::mod_floor(7, -3) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NumberExact(bigF("-2"))),
	)
}

func TestAcc_ModFloor_NegativeNegative(t *testing.T) {
	// -7 mod -3: floor(-7 / -3) = floor(2.333) = 2; -7 - (-3 * 2) = -7 + 6 = -1
	runOutputTest(t,
		`output "test" { value = provider::burnham::mod_floor(-7, -3) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NumberExact(bigF("-1"))),
	)
}

func TestAcc_ModFloor_Fractional(t *testing.T) {
	// 5.5 mod 2 = 1.5
	runOutputTest(t,
		`output "test" { value = provider::burnham::mod_floor(5.5, 2) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NumberExact(bigF("1.5"))),
	)
}

func TestAcc_ModFloor_Zero(t *testing.T) {
	// 6 mod 3 = 0; non-negative result is the canonical case
	runOutputTest(t,
		`output "test" { value = provider::burnham::mod_floor(6, 3) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NumberExact(bigF("0"))),
	)
}

func TestAcc_ModFloor_DivByZero(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::mod_floor(7, 0) }`,
		regexp.MustCompile(`(?is)b\s+must\s+be\s+non-zero`),
	)
}

// ─── clamp ─────────────────────────────────────────────────────────────

func TestAcc_Clamp_InRange(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::clamp(5, 0, 10) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NumberExact(bigF("5"))),
	)
}

func TestAcc_Clamp_BelowMin(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::clamp(-5, 0, 10) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NumberExact(bigF("0"))),
	)
}

func TestAcc_Clamp_AboveMax(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::clamp(99, 0, 10) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NumberExact(bigF("10"))),
	)
}

func TestAcc_Clamp_AtBounds(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::clamp(10, 0, 10) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NumberExact(bigF("10"))),
	)
}

func TestAcc_Clamp_RejectsInvertedBounds(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::clamp(5, 10, 0) }`,
		regexp.MustCompile(`(?is)min_val.*must\s+be\s+<=\s+max_val`),
	)
}
