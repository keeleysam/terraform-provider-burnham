package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// Test fixtures — a self-signed Ed25519 cert and matching CSR generated once with openssl, locked here so the tests don't depend on a usable openssl at test time. The cert is valid 2026-05-08 → 2036-05-05.
const (
	certHeredoc = "<<EOT\n" + testCertPEM + "EOT\n"
	csrHeredoc  = "<<EOT\n" + testCSRPEM + "EOT\n"
)

const (
	testCertPEM = `-----BEGIN CERTIFICATE-----
MIICDTCCAb+gAwIBAgIUMgVGWFtmsm94f67URPQkQej2tdswBQYDK2VwMEMxHTAb
BgNVBAMMFHRlc3QuYnVybmhhbS5leGFtcGxlMRUwEwYDVQQKDAxCdXJuaGFtIFRl
c3QxCzAJBgNVBAYTAlVTMB4XDTI2MDUwODA1NDQ1NFoXDTM2MDUwNTA1NDQ1NFow
QzEdMBsGA1UEAwwUdGVzdC5idXJuaGFtLmV4YW1wbGUxFTATBgNVBAoMDEJ1cm5o
YW0gVGVzdDELMAkGA1UEBhMCVVMwKjAFBgMrZXADIQC/PkuXOt6DOQTVrL6iEgju
7V35EUWLG7lVCTIke/O/F6OBxDCBwTBpBgNVHREEYjBgghR0ZXN0LmJ1cm5oYW0u
ZXhhbXBsZYIYd3d3LnRlc3QuYnVybmhhbS5leGFtcGxlhwTAAAIBgQ9vcHNAZXhh
bXBsZS5jb22GF2h0dHBzOi8vZXhhbXBsZS5jb20vY3NyMAkGA1UdEwQCMAAwCwYD
VR0PBAQDAgWgMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjAdBgNVHQ4E
FgQU2yLvWoHejGQd/33oyjMrnr+Gq5AwBQYDK2VwA0EAfJ8cvzfTb8Y5cC/wcB8H
MKsGUQhtUnEQyWWM8whCWRx4Y2Lfy0gkm8O0hVqtVnpY9YGaI4D5RBvWNLO+cvIH
Bw==
-----END CERTIFICATE-----
`
	testCSRPEM = `-----BEGIN CERTIFICATE REQUEST-----
MIIBezCCAS0CAQAwQzEdMBsGA1UEAwwUdGVzdC5idXJuaGFtLmV4YW1wbGUxFTAT
BgNVBAoMDEJ1cm5oYW0gVGVzdDELMAkGA1UEBhMCVVMwKjAFBgMrZXADIQBdRy86
BPldM5uQC7PkxejuEo3cQV4s7VRFh4IwX/Bjz6CBtjCBswYJKoZIhvcNAQkOMYGl
MIGiMGkGA1UdEQRiMGCCFHRlc3QuYnVybmhhbS5leGFtcGxlghh3d3cudGVzdC5i
dXJuaGFtLmV4YW1wbGWHBMAAAgGBD29wc0BleGFtcGxlLmNvbYYXaHR0cHM6Ly9l
eGFtcGxlLmNvbS9jc3IwCQYDVR0TBAIwADALBgNVHQ8EBAMCBaAwHQYDVR0lBBYw
FAYIKwYBBQUHAwEGCCsGAQUFBwMCMAUGAytlcANBAK9YRCz7VOyYUdbUWsS4uSES
Odr5Q/4/OQM+WhKq/ckVK43GHJ9YSbSnrBrbvHmKMsgvwSMt61Ljyi6fKro4XA0=
-----END CERTIFICATE REQUEST-----
`
)

// ─── hmac (RFC 2104) ───────────────────────────────────────────────────

func TestAcc_HMAC_SHA256RFCTestVector(t *testing.T) {
	// RFC 4231 §4.2 Test Case 1: key = 0x0b * 20, data = "Hi There".
	// HCL strings only support \uNNNN escapes (no \v / \xNN), so we spell the key as twenty `` escapes — each one yields a single 0x0b byte (UTF-8 of U+000B).
	runOutputTest(t,
		`output "test" { value = provider::burnham::hmac("sha256", "", "Hi There") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("b0344c61d8db38535ca8afceaf0bf12b881dc200c9833da726e9376c2e32cff7")),
	)
}

func TestAcc_HMAC_SHA1Length(t *testing.T) {
	// HMAC-SHA-1 output is always 40 hex chars (20 bytes).
	runOutputTest(t,
		`output "test" { value = length(provider::burnham::hmac("sha1", "k", "m")) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Int64Exact(40)),
	)
}

func TestAcc_HMAC_SHA512Length(t *testing.T) {
	// HMAC-SHA-512 output is always 128 hex chars (64 bytes).
	runOutputTest(t,
		`output "test" { value = length(provider::burnham::hmac("sha512", "k", "m")) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Int64Exact(128)),
	)
}

func TestAcc_HMAC_RejectsBadAlgorithm(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::hmac("md5", "k", "m") }`,
		regexp.MustCompile(`(?is)algorithm\s+must\s+be\s+one\s+of`),
	)
}

func TestAcc_HMAC_Determinism(t *testing.T) {
	runOutputTest(t,
		`output "test" {
		   value = provider::burnham::hmac("sha256", "key", "msg") == provider::burnham::hmac("sha256", "key", "msg")
		 }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)),
	)
}

// ─── hkdf (RFC 5869) ───────────────────────────────────────────────────

func TestAcc_HKDF_ASCIIVector(t *testing.T) {
	// Locked-in expected from a Python reference implementation:
	//   hkdf(SHA-256, ikm="secret", salt="salt", info="info", L=42)
	// HCL strings are UTF-8 and don't support \xNN byte escapes, so the canonical RFC 5869 Test Case 1 (which uses high-byte values) can't be expressed verbatim. ASCII-only inputs still verify correctness end-to-end.
	runOutputTest(t,
		`output "test" { value = provider::burnham::hkdf("sha256", "secret", "salt", "info", 42) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("f6d2fcc47cb939deafe3853a1e641a27e6924aff7a63d09cb04ccfffbe4776efdda39ae362b1346092d8")),
	)
}

func TestAcc_HKDF_LengthIsBytesNotHex(t *testing.T) {
	// Length 32 → 64 hex chars.
	runOutputTest(t,
		`output "test" { value = length(provider::burnham::hkdf("sha256", "secret", "salt", "info", 32)) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Int64Exact(64)),
	)
}

func TestAcc_HKDF_RejectsZeroLength(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::hkdf("sha256", "s", "salt", "info", 0) }`,
		regexp.MustCompile(`(?is)length\s+must\s+be\s+>\s+0`),
	)
}

func TestAcc_HKDF_RejectsExcessiveLength(t *testing.T) {
	// SHA-256 produces 32 bytes, so HKDF max is 255 × 32 = 8160. 9999 is over.
	runErrorTest(t,
		`output "test" { value = provider::burnham::hkdf("sha256", "s", "salt", "info", 9999) }`,
		regexp.MustCompile(`(?is)at\s+most\s+255\s+×\s+hashLen`),
	)
}

// ─── pem_decode ─────────────────────────────────────────────────────────

func TestAcc_PEMDecode_SingleCertBlock(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::pem_decode(`+certHeredoc+`)[0].type }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("CERTIFICATE")),
	)
}

func TestAcc_PEMDecode_BlockCount(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = length(provider::burnham::pem_decode(`+certHeredoc+`)) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Int64Exact(1)),
	)
}

func TestAcc_PEMDecode_NoBlocksReturnsEmptyList(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = length(provider::burnham::pem_decode("not a pem block")) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Int64Exact(0)),
	)
}

func TestAcc_PEMDecode_BodyIsBase64(t *testing.T) {
	// Body is base64 of the DER bytes; we just assert it is non-empty and base64-shaped.
	runOutputTest(t,
		`output "test" { value = provider::burnham::pem_decode(`+certHeredoc+`)[0].base64_body }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringRegexp(regexp.MustCompile(`^[A-Za-z0-9+/]+=*$`))),
	)
}

// ─── x509_inspect ───────────────────────────────────────────────────────

func TestAcc_X509Inspect_SubjectAndSANs(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::x509_inspect(`+certHeredoc+`) }`,
		statecheck.ExpectKnownOutputValueAtPath("test",
			tfjsonpath.New("dns_names"),
			knownvalue.ListExact([]knownvalue.Check{
				knownvalue.StringExact("test.burnham.example"),
				knownvalue.StringExact("www.test.burnham.example"),
			}),
		),
	)
}

func TestAcc_X509Inspect_IsNotCA(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::x509_inspect(`+certHeredoc+`).is_ca }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(false)),
	)
}

func TestAcc_X509Inspect_Validity(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::x509_inspect(`+certHeredoc+`).not_before }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("2026-05-08T05:44:54Z")),
	)
}

func TestAcc_X509Inspect_KeyUsage(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::x509_inspect(`+certHeredoc+`).key_usage }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ListExact([]knownvalue.Check{
			knownvalue.StringExact("digitalSignature"),
			knownvalue.StringExact("keyEncipherment"),
		})),
	)
}

func TestAcc_X509Inspect_ExtKeyUsage(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::x509_inspect(`+certHeredoc+`).ext_key_usage }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ListExact([]knownvalue.Check{
			knownvalue.StringExact("serverAuth"),
			knownvalue.StringExact("clientAuth"),
		})),
	)
}

func TestAcc_X509Inspect_RejectsNonCert(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::x509_inspect("not a cert") }`,
		regexp.MustCompile(`(?is)no\s+CERTIFICATE\s+block\s+found`),
	)
}

// ─── x509_fingerprint ───────────────────────────────────────────────────

func TestAcc_X509Fingerprint_SHA256(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::x509_fingerprint(`+certHeredoc+`, "sha256") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("6d2d325a319a26c8d89f417fc543d2673bdce5f9ba2e4ae2bdc6f409f0e346cc")),
	)
}

func TestAcc_X509Fingerprint_SHA1(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::x509_fingerprint(`+certHeredoc+`, "sha1") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("024b4ecf6cd77488ed286191ce1324624bb3b2a8")),
	)
}

func TestAcc_X509Fingerprint_RejectsBadAlgorithm(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::x509_fingerprint(`+certHeredoc+`, "md5") }`,
		regexp.MustCompile(`(?is)algorithm\s+must\s+be`),
	)
}

// ─── csr_inspect ────────────────────────────────────────────────────────

func TestAcc_CSRInspect_DNSNames(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::csr_inspect(`+csrHeredoc+`).dns_names }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ListExact([]knownvalue.Check{
			knownvalue.StringExact("test.burnham.example"),
			knownvalue.StringExact("www.test.burnham.example"),
		})),
	)
}

func TestAcc_CSRInspect_PublicKeyAlgorithm(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::csr_inspect(`+csrHeredoc+`).public_key_algorithm }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("Ed25519")),
	)
}

func TestAcc_CSRInspect_RejectsNonCSR(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::csr_inspect(`+certHeredoc+`) }`,
		regexp.MustCompile(`(?is)no\s+CERTIFICATE\s+REQUEST`),
	)
}

// ─── asn1_decode ────────────────────────────────────────────────────────

func TestAcc_ASN1Decode_SimpleInteger(t *testing.T) {
	// DER for INTEGER 42: 02 01 2a → base64 "AgEq"
	runOutputTest(t,
		`output "test" { value = provider::burnham::asn1_decode("AgEq").value }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("42")),
	)
}

func TestAcc_ASN1Decode_IntegerType(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::asn1_decode("AgEq").type }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("INTEGER")),
	)
}

func TestAcc_ASN1Decode_OID(t *testing.T) {
	// DER for OID 1.2.840.113549.1.1.11 (sha256WithRSAEncryption): 06 09 2a 86 48 86 f7 0d 01 01 0b → base64 "BgkqhkiG9w0BAQs"
	runOutputTest(t,
		`output "test" { value = provider::burnham::asn1_decode("BgkqhkiG9w0BAQs=").value }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("1.2.840.113549.1.1.11")),
	)
}

func TestAcc_ASN1Decode_SequenceOfTwoInts(t *testing.T) {
	// DER: SEQUENCE { INTEGER 1, INTEGER 2 } = 30 06 02 01 01 02 01 02 → base64 "MAYCAQECAQI="
	runOutputTest(t,
		`output "test" { value = length(provider::burnham::asn1_decode("MAYCAQECAQI=").children) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Int64Exact(2)),
	)
}

func TestAcc_ASN1Decode_SequenceCompound(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::asn1_decode("MAYCAQECAQI=").compound }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)),
	)
}

func TestAcc_ASN1Decode_RejectsBadBase64(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::asn1_decode("not!base64") }`,
		regexp.MustCompile(`(?is)valid\s+base64`),
	)
}

func TestAcc_ASN1Decode_RejectsTruncated(t *testing.T) {
	// Header says length 10 but we only have 1 byte: 02 0a 2a → base64 "Agoq"
	runErrorTest(t,
		`output "test" { value = provider::burnham::asn1_decode("Agoq") }`,
		regexp.MustCompile(`(?is)decoding\s+ASN\.1`),
	)
}

func TestAcc_ASN1Decode_AcceptsModerateDepth(t *testing.T) {
	// 8 levels of SEQUENCE wrapping a NULL — well under the 64 depth cap. Verifies the outermost decode succeeds.
	runOutputTest(t,
		`output "test" { value = provider::burnham::asn1_decode("MBAwDjAMMAowCDAGMAQwAgUA").type }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("SEQUENCE")),
	)
}

func TestAcc_ASN1Decode_RejectsExcessiveDepth(t *testing.T) {
	// 70 levels of SEQUENCE — exceeds asn1MaxDepth (64). Hand-built once and locked. Defends against adversarial deep input that would otherwise grow the goroutine stack until the Terraform process OOMs.
	runErrorTest(t,
		`output "test" { value = provider::burnham::asn1_decode("MIGSMIGPMIGMMIGJMIGGMIGDMIGAMH4wfDB6MHgwdjB0MHIwcDBuMGwwajBoMGYwZDBiMGAwXjBcMFowWDBWMFQwUjBQME4wTDBKMEgwRjBEMEIwQDA+MDwwOjA4MDYwNDAyMDAwLjAsMCowKDAmMCQwIjAgMB4wHDAaMBgwFjAUMBIwEDAOMAwwCjAIMAYwBDACBQA=") }`,
		regexp.MustCompile(`(?is)nesting\s+exceeds\s+maximum\s+supported\s+depth`),
	)
}
