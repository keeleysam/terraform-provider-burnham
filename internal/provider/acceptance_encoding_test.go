package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

// ─── hexencode / hexdecode ──────────────────────────────────────

func TestAcc_HexEncode(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::hexencode("Hi") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("4869")),
	)
}

func TestAcc_HexDecode_LenientWhitespaceAndCase(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::hexdecode("48 69") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("Hi")),
	)
}

func TestAcc_HexRoundTrip(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::hexdecode(provider::burnham::hexencode("burnham")) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("burnham")),
	)
}

func TestAcc_HexDecode_Invalid(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::hexdecode("zz") }`,
		regexp.MustCompile(`(?i)invalid hex`),
	)
}

// ─── base64encode / base64decode ────────────────────────────────

func TestAcc_Base64Encode_DefaultMatchesCore(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::base64encode("Hello") == base64encode("Hello") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)),
	)
}

func TestAcc_Base64Encode_URLSafeNoPadding(t *testing.T) {
	// "f8ff" decodes to bytes whose standard base64 contains '/'; url-safe must not.
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::base64encode(provider::burnham::hexdecode("fbffbf"), { url_safe = true, padding = false })
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("-_-_")),
	)
}

func TestAcc_Base64Decode_AcceptsUnpaddedURLSafe(t *testing.T) {
	// "SGVsbG8" is unpadded; url-safe alphabet is a no-op here but exercises the lenient path.
	runOutputTest(t,
		`output "test" { value = provider::burnham::base64decode("SGVsbG8") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("Hello")),
	)
}

func TestAcc_Base64Decode_AcceptsCoreOutput(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::base64decode(base64encode("Hello")) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("Hello")),
	)
}

func TestAcc_Base64_RoundTripURLSafe(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::base64decode(
				provider::burnham::base64encode("burnham/plan?ok", { url_safe = true, padding = false })
			)
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("burnham/plan?ok")),
	)
}

func TestAcc_Base64Encode_RejectsUnknownOption(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::base64encode("x", { wrap = 76 }) }`,
		regexp.MustCompile(`(?i)unknown option key`),
	)
}

// ─── base32encode / base32decode ────────────────────────────────

func TestAcc_Base32Encode_RFCVector(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::base32encode("foobar") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("MZXW6YTBOI======")),
	)
}

func TestAcc_Base32Encode_HexAlphabetNoPadding(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::base32encode("foobar", { hex_alphabet = true, padding = false }) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("CPNMUOJ1E8")),
	)
}

func TestAcc_Base32Decode_LenientCaseAndPadding(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::base32decode("mzxw6ytboi") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("foobar")),
	)
}

func TestAcc_Base32_RoundTrip(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::base32decode(provider::burnham::base32encode("burnham"))
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("burnham")),
	)
}

func TestAcc_Base32Decode_Invalid(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::base32decode("0189") }`,
		regexp.MustCompile(`(?i)invalid base32`),
	)
}

// ─── urlencode / urldecode ──────────────────────────────────────

func TestAcc_URLEncode_DefaultMatchesCore(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::urlencode("a b/c") == urlencode("a b/c") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)),
	)
}

func TestAcc_URLEncode_PathMode(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::urlencode("a b/c", { mode = "path" }) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("a%20b%2Fc")),
	)
}

func TestAcc_URLDecode_QueryPlusIsSpace(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::urldecode("a+b%2Fc") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("a b/c")),
	)
}

func TestAcc_URLDecode_PathPlusIsLiteral(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::urldecode("1+1", { mode = "path" }) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("1+1")),
	)
}

func TestAcc_URLDecode_Invalid(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::urldecode("%ZZ") }`,
		regexp.MustCompile(`(?i)invalid URL-encoded`),
	)
}

func TestAcc_URLEncode_RejectsBadMode(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::urlencode("x", { mode = "raw" }) }`,
		regexp.MustCompile(`(?i)mode must be one of`),
	)
}
