package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

// ip_idunno_* (RFC 8771) tests live in their own file alongside the pigeon_throughput tests for the same reason: kept out of the standard CIDR / IP / NAT64 / NPTv6 test surface so the dense reading there isn't diluted by the RFC's idiosyncratic vector.

// TestAcc_IPIDunnoEncode_RFCExample locks the worked example from RFC 8771 §5: encoding `198.51.100.164` must produce exactly the four codepoints the RFC names (U+0063, U+000C, U+006C, U+04A4). Any change to the encoder's layout-priority order that breaks this is a regression against the RFC's only published worked example.
func TestAcc_IPIDunnoEncode_RFCExample(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::ip_idunno_encode("198.51.100.164") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("clҤ")),
	)
}

func TestAcc_IPIDunnoEncode_IPv4Deterministic(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::ip_idunno_encode("10.0.0.1") == provider::burnham::ip_idunno_encode("10.0.0.1") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)),
	)
}

func TestAcc_IPIDunnoEncode_IPv6Deterministic(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::ip_idunno_encode("2001:db8::1") == provider::burnham::ip_idunno_encode("2001:db8::1") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)),
	)
}

func TestAcc_IPIDunnoEncode_DifferentIPsDiffer(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::ip_idunno_encode("10.0.0.1") == provider::burnham::ip_idunno_encode("10.0.0.2") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(false)),
	)
}

func TestAcc_IPIDunnoDecode_RFCExampleRoundTrip(t *testing.T) {
	// Decode the exact RFC §5 byte sequence back to its IPv4 address.
	runOutputTest(t,
		`output "test" { value = provider::burnham::ip_idunno_decode("clҤ") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("198.51.100.164")),
	)
}

func TestAcc_IPIDunnoRoundTrip_IPv4Set(t *testing.T) {
	// Sample IPv4 addresses including boundary cases. Each must round-trip through encode→decode losslessly. The all-zero, last-nibble-zero, and all-ones addresses exercise the layout-search fallback paths.
	cases := []string{"0.0.0.0", "127.0.0.1", "1.2.3.0", "192.168.1.16", "255.255.255.255", "198.51.100.164"}
	for _, ip := range cases {
		t.Run(ip, func(t *testing.T) {
			runOutputTest(t,
				`output "test" { value = provider::burnham::ip_idunno_decode(provider::burnham::ip_idunno_encode("`+ip+`")) }`,
				statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact(ip)),
			)
		})
	}
}

func TestAcc_IPIDunnoRoundTrip_IPv6Set(t *testing.T) {
	// IPv6 addresses canonicalised through the parser. `::ffff:1.2.3.4` would Unmap to its v4 form, so we don't include it here.
	cases := map[string]string{
		"::":          "::",
		"::1":         "::1",
		"2001:db8::1": "2001:db8::1",
		"fe80::1":     "fe80::1",
	}
	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			runOutputTest(t,
				`output "test" { value = provider::burnham::ip_idunno_decode(provider::burnham::ip_idunno_encode("`+in+`")) }`,
				statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact(want)),
			)
		})
	}
}

func TestAcc_IPIDunnoEncode_RejectsInvalidIP(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::ip_idunno_encode("not-an-ip") }`,
		regexp.MustCompile(`(?is)invalid\s+IP\s+address`),
	)
}

func TestAcc_IPIDunnoDecode_RejectsEmptyInput(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::ip_idunno_decode("") }`,
		regexp.MustCompile(`(?is)input\s+must\s+not\s+be\s+empty`),
	)
}

func TestAcc_IPIDunnoDecode_RejectsOutOfRangePayload(t *testing.T) {
	// Three ASCII codepoints = 3 × 7 = 21 bits of payload, which falls in neither [32, 52] nor [128, 148].
	runErrorTest(t,
		`output "test" { value = provider::burnham::ip_idunno_decode("abc") }`,
		regexp.MustCompile(`(?is)total\s+bit-payload\s+21\s+does\s+not\s+match`),
	)
}
