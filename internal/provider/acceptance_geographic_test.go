package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

// ─── geohash_encode ─────────────────────────────────────────────────────

func TestAcc_GeohashEncode_SanFranciscoCivicCenter(t *testing.T) {
	// (37.7749, -122.4194) at precision 7 yields "9q8yyk8" — a stable, well-known reference value.
	runOutputTest(t,
		`output "test" { value = provider::burnham::geohash_encode(37.7749, -122.4194, 7) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("9q8yyk8")),
	)
}

func TestAcc_GeohashEncode_PrecisionOne(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = length(provider::burnham::geohash_encode(0, 0, 1)) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Int64Exact(1)),
	)
}

func TestAcc_GeohashEncode_PrecisionTwelve(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = length(provider::burnham::geohash_encode(0, 0, 12)) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Int64Exact(12)),
	)
}

func TestAcc_GeohashEncode_RejectsExcessivePrecision(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::geohash_encode(0, 0, 15) }`,
		regexp.MustCompile(`(?is)precision\s+must\s+be\s+in\s+\[1,\s*12\]`),
	)
}

func TestAcc_GeohashEncode_RejectsOutOfRangeLatitude(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::geohash_encode(91, 0, 5) }`,
		regexp.MustCompile(`(?is)latitude\s+must\s+be\s+in\s+\[-90,\s*90\]`),
	)
}

func TestAcc_GeohashEncode_RejectsOutOfRangeLongitude(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::geohash_encode(0, 181, 5) }`,
		regexp.MustCompile(`(?is)longitude\s+must\s+be\s+in\s+\[-180,\s*180\]`),
	)
}

func TestAcc_GeohashEncode_RejectsCornerLatitude90(t *testing.T) {
	// Upstream wraps lat==90 to lat==-90; we reject so callers don't silently get the opposite quadrant.
	runErrorTest(t,
		`output "test" { value = provider::burnham::geohash_encode(90, 0, 5) }`,
		regexp.MustCompile(`(?is)latitude\s+==\s+90\s+wraps`),
	)
}

func TestAcc_GeohashEncode_RejectsCornerLongitude180(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::geohash_encode(0, 180, 5) }`,
		regexp.MustCompile(`(?is)longitude\s+==\s+180\s+wraps`),
	)
}

func TestAcc_GeohashDecode_CornerCellBBoxEdgesRoundTrip(t *testing.T) {
	// Regression: previously the decoder reported `lat_max=90, lon_max=180` for "zzzzzzzzzzzz", which the encoder rejects. The decoder now shrinks those edges below the wrap threshold so feeding them back into `geohash_encode` lands on the same corner cell (prefix "z") instead of erroring.
	runOutputTest(t,
		`output "test" { value = substr(provider::burnham::geohash_encode(provider::burnham::geohash_decode("zzzzzzzzzzzz").lat_max, provider::burnham::geohash_decode("zzzzzzzzzzzz").lon_max, 1), 0, 1) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("z")),
	)
}

// ─── geohash_decode ─────────────────────────────────────────────────────

func TestAcc_GeohashDecode_RoundTripCenter(t *testing.T) {
	// Decoding "9q8yyk8" should give back coordinates close to (37.7749, -122.4194). We allow some slack via the bounding box rather than requiring exact float equality.
	runOutputTest(t,
		`output "test" {
		   value = (
		     provider::burnham::geohash_decode("9q8yyk8").lat_min < 37.7749 &&
		     provider::burnham::geohash_decode("9q8yyk8").lat_max > 37.7749 &&
		     provider::burnham::geohash_decode("9q8yyk8").lon_min < -122.4194 &&
		     provider::burnham::geohash_decode("9q8yyk8").lon_max > -122.4194
		   )
		 }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)),
	)
}

func TestAcc_GeohashDecode_CaseInsensitive(t *testing.T) {
	// Upper-case input should decode to the same centre as lower-case.
	runOutputTest(t,
		`output "test" {
		   value = (
		     provider::burnham::geohash_decode("9Q8YYK8").latitude == provider::burnham::geohash_decode("9q8yyk8").latitude
		   )
		 }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)),
	)
}

func TestAcc_GeohashDecode_RejectsBadAlphabet(t *testing.T) {
	// 'a', 'i', 'l', 'o' are not in the geohash alphabet.
	runErrorTest(t,
		`output "test" { value = provider::burnham::geohash_decode("aaaa") }`,
		regexp.MustCompile(`(?is)invalid\s+character`),
	)
}

func TestAcc_GeohashDecode_RejectsEmpty(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::geohash_decode("") }`,
		regexp.MustCompile(`(?is)code\s+must\s+not\s+be\s+empty`),
	)
}

// ─── pluscode_encode ────────────────────────────────────────────────────

func TestAcc_PluscodeEncode_SanFranciscoCivicCenter(t *testing.T) {
	// (37.7749, -122.4194) at length 10 yields the canonical "849VQHFJ+X6" cross-checked against Google's OLC reference implementation.
	runOutputTest(t,
		`output "test" { value = provider::burnham::pluscode_encode(37.7749, -122.4194, 10) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("849VQHFJ+X6")),
	)
}

func TestAcc_PluscodeEncode_HasPlus(t *testing.T) {
	// Every full Plus code contains "+" between characters 8 and 9.
	runOutputTest(t,
		`output "test" {
		   value = substr(provider::burnham::pluscode_encode(37.7749, -122.4194, 10), 8, 1)
		 }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("+")),
	)
}

func TestAcc_PluscodeEncode_RejectsOddLengthBelow11(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::pluscode_encode(0, 0, 7) }`,
		regexp.MustCompile(`(?is)length\s+in\s+\[2,\s*10\]\s+must\s+be\s+even`),
	)
}

func TestAcc_PluscodeEncode_AcceptsLength11(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = length(provider::burnham::pluscode_encode(0, 0, 11)) }`,
		// Code is "AAAAAAAA+AAA" or similar — 11 base characters + 1 "+" = 12 total.
		statecheck.ExpectKnownOutputValue("test", knownvalue.Int64Exact(12)),
	)
}

func TestAcc_PluscodeEncode_RejectsExcessiveLength(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::pluscode_encode(0, 0, 16) }`,
		regexp.MustCompile(`(?is)length\s+must\s+be\s+in\s+\[2,\s*15\]`),
	)
}

// ─── pluscode_decode ────────────────────────────────────────────────────

func TestAcc_PluscodeDecode_RoundTrip(t *testing.T) {
	// Encode then decode and confirm the centre is inside the original cell.
	runOutputTest(t,
		`output "test" {
		   value = (
		     provider::burnham::pluscode_decode(provider::burnham::pluscode_encode(37.7749, -122.4194, 10)).lat_min < 37.7749 &&
		     provider::burnham::pluscode_decode(provider::burnham::pluscode_encode(37.7749, -122.4194, 10)).lat_max > 37.7749 &&
		     provider::burnham::pluscode_decode(provider::burnham::pluscode_encode(37.7749, -122.4194, 10)).lon_min < -122.4194 &&
		     provider::burnham::pluscode_decode(provider::burnham::pluscode_encode(37.7749, -122.4194, 10)).lon_max > -122.4194
		   )
		 }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)),
	)
}

func TestAcc_PluscodeDecode_LengthFieldMatchesEncoder(t *testing.T) {
	runOutputTest(t,
		`output "test" {
		   value = provider::burnham::pluscode_decode(provider::burnham::pluscode_encode(0, 0, 8)).length
		 }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Int64Exact(8)),
	)
}

func TestAcc_PluscodeDecode_RejectsBadCode(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::pluscode_decode("not-a-pluscode") }`,
		regexp.MustCompile(`(?is)full\s+Open\s+Location\s+Code`),
	)
}
