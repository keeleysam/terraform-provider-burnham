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
