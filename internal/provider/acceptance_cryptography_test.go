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

// Test fixture for pkcs7_sign's "external identity" mode — a real ECDSA P-256 keypair generated once with `openssl ecparam -name prime256v1 -genkey | openssl pkcs8 -topk8 -nocrypt` and a self-signed cert for it. Locked here so acceptance tests don't shell out to openssl at test time. CN=burnham-test-p256, validity 2026 → 2036.
const (
	testECDSAP256KeyPEM = `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQg7zF1t2VFJJWPOcHi
0BYZnkWB/2bOBBXxWzMtnATpn06hRANCAARjCS1S5sK75BKMvJR1m7YEPEupniYQ
6tG3M2IpxmIhPg9nGX5lXzVid74I+RtkAIZmz6UmGFMGnEP1orEig3Nm
-----END PRIVATE KEY-----
`
	testECDSAP256CertPEM = `-----BEGIN CERTIFICATE-----
MIIByzCCAXGgAwIBAgIUMASm5ZLV8Bfy8XpJVbvyJqBCoAQwCgYIKoZIzj0EAwIw
OzEaMBgGA1UEAwwRYnVybmhhbS10ZXN0LXAyNTYxEDAOBgNVBAoMB0J1cm5oYW0x
CzAJBgNVBAYTAlVTMB4XDTI2MDUyMTAyMTg1N1oXDTM2MDUxODAyMTg1N1owOzEa
MBgGA1UEAwwRYnVybmhhbS10ZXN0LXAyNTYxEDAOBgNVBAoMB0J1cm5oYW0xCzAJ
BgNVBAYTAlVTMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEYwktUubCu+QSjLyU
dZu2BDxLqZ4mEOrRtzNiKcZiIT4PZxl+ZV81Yne+CPkbZACGZs+lJhhTBpxD9aKx
IoNzZqNTMFEwHQYDVR0OBBYEFEFT2fsexjM6+I4yyoxLMxwU2AFIMB8GA1UdIwQY
MBaAFEFT2fsexjM6+I4yyoxLMxwU2AFIMA8GA1UdEwEB/wQFMAMBAf8wCgYIKoZI
zj0EAwIDSAAwRQIgEhguARQtpGPMrwtGvn5ak8g7MhrYdG7xLbJm7wROeP0CIQCL
2CMof37FZZUJUktQ53AeeaKNJ1EaIa+6J1GskqhWdg==
-----END CERTIFICATE-----
`
	testECDSAP256KeyHeredoc  = "<<EOT\n" + testECDSAP256KeyPEM + "EOT\n"
	testECDSAP256CertHeredoc = "<<EOT\n" + testECDSAP256CertPEM + "EOT\n"
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

func TestAcc_PEMDecode_RejectsOversizedInput(t *testing.T) {
	// 16 MiB + 1 byte exceeds pemMaxInputBytes; the function rejects before walking pem.Decode.
	runErrorTest(t,
		`output "test" { value = provider::burnham::pem_decode(format("%-16777217s", " ")) }`,
		regexp.MustCompile(`(?is)pem\s+input\s+exceeds\s+maximum\s+supported\s+length`),
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

func TestAcc_X509Inspect_NotAfter(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::x509_inspect(`+certHeredoc+`).not_after }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("2036-05-05T05:44:54Z")),
	)
}

func TestAcc_X509Inspect_Subject(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::x509_inspect(`+certHeredoc+`).subject }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("CN=test.burnham.example,O=Burnham Test,C=US")),
	)
}

func TestAcc_X509Inspect_PublicKeyAlgorithm(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::x509_inspect(`+certHeredoc+`).public_key_algorithm }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("Ed25519")),
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

func TestAcc_X509Fingerprint_SHA384(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::x509_fingerprint(`+certHeredoc+`, "sha384") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("13b45fa66be9626d264b7560cbb8a1bc15f189ef9b90314a456cf5dba534813202c176756974ee80073b89cce20c6352")),
	)
}

func TestAcc_X509Fingerprint_SHA512(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::x509_fingerprint(`+certHeredoc+`, "sha512") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("5fd3798b18fb55e7ba9ad6400a079eca7113646382ad50b95a3f83741981be72d925a43e3938661541c40dcb2099b9556d2e7f2fd6e24a949daa8271dc0c551b")),
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

func TestAcc_ASN1Decode_RejectsOversizedInput(t *testing.T) {
	// 8 MiB + 1 byte exceeds asn1MaxBase64Bytes. Built procedurally with `format("%-N s", " ")` so the test source stays small; the function rejects on length before attempting base64 decode.
	runErrorTest(t,
		`output "test" { value = provider::burnham::asn1_decode(format("%-8388609s", " ")) }`,
		regexp.MustCompile(`(?is)der_base64\s+input\s+exceeds\s+maximum\s+length`),
	)
}

// Primitive-tag coverage. Each fixture is a single-TLV DER that exercises one of decodePrimitive's per-tag branches; pre-computed via encoding/asn1 once and locked. Together they cover BIT STRING, OCTET STRING, UTF8String, PrintableString, BOOLEAN.

func TestAcc_ASN1Decode_BitString(t *testing.T) {
	// BIT STRING { 0x86 } — DER 03 02 00 86 → base64 "AwIAhg==". Decoded value is the hex of the data bytes (the leading unused-bits octet is stripped by encoding/asn1.BitString).
	runOutputTest(t,
		`output "test" { value = provider::burnham::asn1_decode("AwIAhg==").value }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("86")),
	)
}

func TestAcc_ASN1Decode_OctetString(t *testing.T) {
	// OCTET STRING "hello" — DER 04 05 68 65 6c 6c 6f → base64 "BAVoZWxsbw==". Value is hex of the bytes.
	runOutputTest(t,
		`output "test" { value = provider::burnham::asn1_decode("BAVoZWxsbw==").value }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("68656c6c6f")),
	)
}

func TestAcc_ASN1Decode_UTF8String(t *testing.T) {
	// UTF8String "hello" — DER 0c 05 68 65 6c 6c 6f → base64 "DAVoZWxsbw==". Value is the string itself.
	runOutputTest(t,
		`output "test" { value = provider::burnham::asn1_decode("DAVoZWxsbw==").value }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("hello")),
	)
}

func TestAcc_ASN1Decode_BooleanTrue(t *testing.T) {
	// BOOLEAN true — DER 01 01 ff → base64 "AQH/".
	runOutputTest(t,
		`output "test" { value = provider::burnham::asn1_decode("AQH/").value }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("true")),
	)
}

func TestAcc_ASN1Decode_BooleanFalse(t *testing.T) {
	// BOOLEAN false — DER 01 01 00 → base64 "AQEA".
	runOutputTest(t,
		`output "test" { value = provider::burnham::asn1_decode("AQEA").value }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("false")),
	)
}

func TestAcc_ASN1Decode_BMPStringDecodesUTF16(t *testing.T) {
	// BMPString "Hi" — DER 1e 04 00 48 00 69 → base64 "HgQASABp". Regression: previously this returned the raw UCS-2BE bytes as a Go string (mojibake); now it decodes to UTF-8 via utf16.Decode so the value is a real string.
	runOutputTest(t,
		`output "test" { value = provider::burnham::asn1_decode("HgQASABp").value }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("Hi")),
	)
}

// ─── ecdsa_p256_key_from_seed ──────────────────────────────────────────

func TestAcc_ECDSAP256KeyFromSeed_ReturnsPKCS8PEM(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::ecdsa_p256_key_from_seed("burnham-test") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringRegexp(regexp.MustCompile(`(?s)^-----BEGIN PRIVATE KEY-----\n.+\n-----END PRIVATE KEY-----\n$`))),
	)
}

func TestAcc_ECDSAP256KeyFromSeed_DeterministicSameSeed(t *testing.T) {
	// Two derivations from the same seed must produce the same PEM. Compares them inside a single plan — the function being non-deterministic would surface as a diff equal-check failure.
	runOutputTest(t,
		`output "test" { value = provider::burnham::ecdsa_p256_key_from_seed("same-seed") == provider::burnham::ecdsa_p256_key_from_seed("same-seed") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)),
	)
}

func TestAcc_ECDSAP256KeyFromSeed_DifferentSeedsDiffer(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::ecdsa_p256_key_from_seed("seed-a") == provider::burnham::ecdsa_p256_key_from_seed("seed-b") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(false)),
	)
}

func TestAcc_ECDSAP256KeyFromSeed_RejectsEmptySeed(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::ecdsa_p256_key_from_seed("") }`,
		regexp.MustCompile(`(?is)seed\s+must\s+not\s+be\s+empty`),
	)
}

// ─── x509_self_sign ─────────────────────────────────────────────────────

func TestAcc_X509SelfSign_ProducesParseableCert(t *testing.T) {
	// Chain ecdsa_p256_key_from_seed → x509_self_sign → x509_inspect: confirms the cert burnham emits parses back through burnham's own inspector. CN round-trip is the smoking-gun field — if cert assembly broke, this would be empty or wrong.
	config := `
locals {
  key  = provider::burnham::ecdsa_p256_key_from_seed("acc-test")
  cert = provider::burnham::x509_self_sign(local.key, "burnham.acc", "deterministic-serial", "2001-01-01T00:00:00Z", "2099-01-01T00:00:00Z")
}
output "test" { value = provider::burnham::x509_inspect(local.cert).subject }
`
	runOutputTest(t,
		config,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("CN=burnham.acc")),
	)
}

func TestAcc_X509SelfSign_Deterministic(t *testing.T) {
	config := `
locals {
  key = provider::burnham::ecdsa_p256_key_from_seed("det")
  a   = provider::burnham::x509_self_sign(local.key, "burnham.det", "serial-15-bytes", "2001-01-01T00:00:00Z", "2099-01-01T00:00:00Z")
  b   = provider::burnham::x509_self_sign(local.key, "burnham.det", "serial-15-bytes", "2001-01-01T00:00:00Z", "2099-01-01T00:00:00Z")
}
output "test" { value = local.a == local.b }
`
	runOutputTest(t, config, statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)))
}

func TestAcc_X509SelfSign_RejectsBadValidity(t *testing.T) {
	config := `
locals { key = provider::burnham::ecdsa_p256_key_from_seed("seed") }
output "test" { value = provider::burnham::x509_self_sign(local.key, "cn", "serial", "2099-01-01T00:00:00Z", "2001-01-01T00:00:00Z") }
`
	runErrorTest(t, config, regexp.MustCompile(`(?is)not_after\s+must\s+be\s+strictly\s+after\s+not_before`))
}

func TestAcc_X509SelfSign_RejectsBadCommonName(t *testing.T) {
	config := `
locals { key = provider::burnham::ecdsa_p256_key_from_seed("seed") }
output "test" { value = provider::burnham::x509_self_sign(local.key, "", "serial", "2001-01-01T00:00:00Z", "2099-01-01T00:00:00Z") }
`
	runErrorTest(t, config, regexp.MustCompile(`(?is)common_name\s+must\s+be\s+1.64`))
}

func TestAcc_X509SelfSign_Rejects65CharCommonName(t *testing.T) {
	// 65 ASCII characters — one over the RFC 5280 §A.1 ub-common-name-length cap. Confirms the upper bound actually fires (the empty-string test above wouldn't catch a regression in the upper-bound branch).
	config := `
locals {
  key = provider::burnham::ecdsa_p256_key_from_seed("seed")
  cn  = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
}
output "test" { value = provider::burnham::x509_self_sign(local.key, local.cn, "serial", "2001-01-01T00:00:00Z", "2099-01-01T00:00:00Z") }
`
	runErrorTest(t, config, regexp.MustCompile(`(?is)common_name\s+must\s+be\s+1.64`))
}

func TestAcc_X509SelfSign_AcceptsMultiByteCommonName(t *testing.T) {
	// 64 CJK characters = 192 UTF-8 bytes. The RFC 5280 §A.1 ub-common-name-length cap is on *characters* (ASN.1 string elements / Unicode code points), not bytes — this CN is RFC-legal and our rune-counting check must accept it. A byte-counting regression would reject this with the "must be 1-64" message.
	// 64 copies of U+4E00 (一, the simplest CJK character) spelled out as \u escapes so the literal stays editable.
	const cn = "一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一一"
	config := `
locals {
  key  = provider::burnham::ecdsa_p256_key_from_seed("multibyte-cn")
  cert = provider::burnham::x509_self_sign(local.key, "` + cn + `", "serial-15-bytes", "2001-01-01T00:00:00Z", "2099-01-01T00:00:00Z")
}
output "test" { value = startswith(local.cert, "-----BEGIN CERTIFICATE-----") }
`
	runOutputTest(t, config, statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)))
}

// ─── pkcs7_sign ─────────────────────────────────────────────────────────

func TestAcc_PKCS7Sign_ASN1DecodeNavigatesNestedHeterogeneous(t *testing.T) {
	// End-to-end shape test for pkcs7_sign output: derive identity → self-sign cert → CMS-sign payload → decode via asn1_decode and assert structural fields at multiple levels of nesting.
	//
	// Subsumes the older "output is base64 DER" smoke check by demanding more: the asn1_decode children-as-tuple fix in this PR is the gate that lets us walk CMS SignedData (heterogeneous SET children — mix of SEQUENCE / OCTET STRING / [0]-tagged blobs at multiple levels). Just checking root `.type` would only prove the outer tag; this navigates two levels deeper to assert heterogeneous-children navigation actually works.
	//
	// CMS ContentInfo SEQUENCE → children[1] is the [0]-EXPLICIT-tagged content (class "context"), which wraps the SignedData SEQUENCE. The wrapped SignedData's first child is the version INTEGER. So `children[1].children[0].children[0].value` is the SignedData version field — per RFC 5652 §5.1 that's "1" for id-data + IssuerAndSerialNumber + plain-cert configurations.
	config := `
locals {
  key     = provider::burnham::ecdsa_p256_key_from_seed("nested-nav-test")
  cert    = provider::burnham::x509_self_sign(local.key, "burnham.nested", "serial-15-bytes", "2001-01-01T00:00:00Z", "2099-01-01T00:00:00Z")
  signed  = provider::burnham::pkcs7_sign("hello, world", local.key, local.cert)
  decoded = provider::burnham::asn1_decode(local.signed)
}
output "outer_type" { value = local.decoded.type }
output "content_class" { value = local.decoded.children[1].class }
output "signed_data_version" { value = local.decoded.children[1].children[0].children[0].value }
`
	runOutputTest(t, config,
		statecheck.ExpectKnownOutputValue("outer_type", knownvalue.StringExact("SEQUENCE")),
		statecheck.ExpectKnownOutputValue("content_class", knownvalue.StringExact("context")),
		statecheck.ExpectKnownOutputValue("signed_data_version", knownvalue.StringExact("1")),
	)
}

func TestAcc_PKCS7Sign_OutputMatchesBase64Charset(t *testing.T) {
	config := `
locals {
  key  = provider::burnham::ecdsa_p256_key_from_seed("charset")
  cert = provider::burnham::x509_self_sign(local.key, "cn", "serial", "2001-01-01T00:00:00Z", "2099-01-01T00:00:00Z")
}
output "test" { value = provider::burnham::pkcs7_sign("payload", local.key, local.cert) }
`
	runOutputTest(t, config, statecheck.ExpectKnownOutputValue("test",
		knownvalue.StringRegexp(regexp.MustCompile(`^[A-Za-z0-9+/]+=*$`))))
}

func TestAcc_PKCS7Sign_Deterministic(t *testing.T) {
	config := `
locals {
  key  = provider::burnham::ecdsa_p256_key_from_seed("det")
  cert = provider::burnham::x509_self_sign(local.key, "cn", "serial", "2001-01-01T00:00:00Z", "2099-01-01T00:00:00Z")
  a    = provider::burnham::pkcs7_sign("payload", local.key, local.cert)
  b    = provider::burnham::pkcs7_sign("payload", local.key, local.cert)
}
output "test" { value = local.a == local.b }
`
	runOutputTest(t, config, statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)))
}

func TestAcc_PKCS7Sign_RejectsEmptyData(t *testing.T) {
	config := `
locals {
  key  = provider::burnham::ecdsa_p256_key_from_seed("seed")
  cert = provider::burnham::x509_self_sign(local.key, "cn", "serial", "2001-01-01T00:00:00Z", "2099-01-01T00:00:00Z")
}
output "test" { value = provider::burnham::pkcs7_sign("", local.key, local.cert) }
`
	runErrorTest(t, config, regexp.MustCompile(`(?is)data\s+must\s+not\s+be\s+empty`))
}

func TestAcc_PKCS7Sign_ExternalIdentity(t *testing.T) {
	// Drive pkcs7_sign with a hardcoded ECDSA P-256 key + cert (the "real identity" mode the function's doc advertises) instead of the derive-from-seed chain. Verifies the PEM-parsing path for both inputs and confirms the signing function works without our derivation primitives. The output is base64-encoded DER; checking it parses as ASN.1 SEQUENCE confirms the function succeeded end-to-end (errors would surface as the function failing before the asn1_decode call).
	config := `
locals {
  signed = provider::burnham::pkcs7_sign("hello", ` + testECDSAP256KeyHeredoc + `, ` + testECDSAP256CertHeredoc + `)
}
output "test" { value = provider::burnham::asn1_decode(local.signed).type }
`
	runOutputTest(t, config, statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("SEQUENCE")))
}

func TestAcc_PKCS7Sign_ExternalIdentityIsDeterministic(t *testing.T) {
	// Even with an externally-supplied identity (where the cert assembly happened outside Terraform with random k), the CMS layer must still produce identical bytes across runs given the same (data, key, cert) — that's the RFC 6979 wrapper doing its job at sign time.
	config := `
locals {
  a = provider::burnham::pkcs7_sign("payload", ` + testECDSAP256KeyHeredoc + `, ` + testECDSAP256CertHeredoc + `)
  b = provider::burnham::pkcs7_sign("payload", ` + testECDSAP256KeyHeredoc + `, ` + testECDSAP256CertHeredoc + `)
}
output "test" { value = local.a == local.b }
`
	runOutputTest(t, config, statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)))
}

func TestAcc_PKCS7Sign_RejectsMismatchedKeyAndCert(t *testing.T) {
	// Key A vs cert for key B — should error rather than silently produce unverifiable CMS.
	config := `
locals { key_b = provider::burnham::ecdsa_p256_key_from_seed("different-seed") }
output "test" { value = provider::burnham::pkcs7_sign("payload", local.key_b, ` + testECDSAP256CertHeredoc + `) }
`
	runErrorTest(t, config, regexp.MustCompile(`(?is)cert_pem\s+public\s+key\s+does\s+not\s+match`))
}

func TestAcc_PKCS7Sign_RejectsOversizedData(t *testing.T) {
	// pkcs7DataMaxBytes is 16 MiB. Use format("%-N s", " ") to procedurally build a 17-MiB string without bloating the test source.
	config := `
locals {
  data = format("%-17825793s", " ")
  key  = provider::burnham::ecdsa_p256_key_from_seed("seed")
  cert = provider::burnham::x509_self_sign(local.key, "cn", "serial", "2001-01-01T00:00:00Z", "2099-01-01T00:00:00Z")
}
output "test" { value = provider::burnham::pkcs7_sign(local.data, local.key, local.cert) }
`
	runErrorTest(t, config, regexp.MustCompile(`(?is)data\s+exceeds\s+maximum\s+length`))
}

func TestAcc_ECDSAP256KeyFromSeed_RejectsOversizedSeed(t *testing.T) {
	// signingSeedMaxBytes is 8 MiB. Use format("%-N s", " ") to procedurally build a >8 MiB string. Confirms the cap fires before the seed reaches HKDF.
	config := `
output "test" { value = provider::burnham::ecdsa_p256_key_from_seed(format("%-8388609s", " ")) }
`
	runErrorTest(t, config, regexp.MustCompile(`(?is)seed\s+exceeds\s+maximum\s+length`))
}

// ─── ed25519_key_from_seed ──────────────────────────────────────────────

func TestAcc_Ed25519KeyFromSeed_ReturnsPKCS8PEM(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::ed25519_key_from_seed("burnham-ed25519-test") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringRegexp(regexp.MustCompile(`(?s)^-----BEGIN PRIVATE KEY-----\n.+\n-----END PRIVATE KEY-----\n$`))),
	)
}

func TestAcc_Ed25519KeyFromSeed_DeterministicSameSeed(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::ed25519_key_from_seed("same-seed") == provider::burnham::ed25519_key_from_seed("same-seed") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)),
	)
}

func TestAcc_Ed25519KeyFromSeed_DifferentSeedsDiffer(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::ed25519_key_from_seed("seed-a") == provider::burnham::ed25519_key_from_seed("seed-b") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(false)),
	)
}

func TestAcc_Ed25519KeyFromSeed_DiffersFromECDSAForSameSeed(t *testing.T) {
	// Same seed bytes, different derivation algorithm + HKDF info string + output format → outputs must not collide. Guards against a refactor that accidentally collapses the two derivation paths.
	runOutputTest(t,
		`output "test" { value = provider::burnham::ecdsa_p256_key_from_seed("seed") == provider::burnham::ed25519_key_from_seed("seed") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(false)),
	)
}

func TestAcc_Ed25519KeyFromSeed_RejectsEmptySeed(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::ed25519_key_from_seed("") }`,
		regexp.MustCompile(`(?is)seed\s+must\s+not\s+be\s+empty`),
	)
}

// ─── x509_self_sign / pkcs7_sign — Ed25519 paths ────────────────────────

func TestAcc_X509SelfSign_AcceptsEd25519Key(t *testing.T) {
	// Chain ed25519_key_from_seed → x509_self_sign → x509_inspect to assert the cert parses and the public-key algorithm comes back as Ed25519.
	config := `
locals {
  key  = provider::burnham::ed25519_key_from_seed("acc-ed25519")
  cert = provider::burnham::x509_self_sign(local.key, "burnham.ed25519", "deterministic-serial", "2001-01-01T00:00:00Z", "2099-01-01T00:00:00Z")
}
output "subject"   { value = provider::burnham::x509_inspect(local.cert).subject }
output "algorithm" { value = provider::burnham::x509_inspect(local.cert).public_key_algorithm }
`
	runOutputTest(t, config,
		statecheck.ExpectKnownOutputValue("subject", knownvalue.StringExact("CN=burnham.ed25519")),
		statecheck.ExpectKnownOutputValue("algorithm", knownvalue.StringExact("Ed25519")),
	)
}

func TestAcc_X509SelfSign_DeterministicEd25519(t *testing.T) {
	config := `
locals {
  key = provider::burnham::ed25519_key_from_seed("det")
  a   = provider::burnham::x509_self_sign(local.key, "burnham.det", "serial-15-bytes", "2001-01-01T00:00:00Z", "2099-01-01T00:00:00Z")
  b   = provider::burnham::x509_self_sign(local.key, "burnham.det", "serial-15-bytes", "2001-01-01T00:00:00Z", "2099-01-01T00:00:00Z")
}
output "test" { value = local.a == local.b }
`
	runOutputTest(t, config, statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)))
}

func TestAcc_PKCS7Sign_Ed25519EndToEnd(t *testing.T) {
	// End-to-end Ed25519 chain: derive identity → self-sign cert → CMS-sign payload → asn1_decode to confirm the SignedData carries the SHA-512 digest algorithm OID in DigestAlgorithms per RFC 8419 §3.
	//
	// CMS structure (ContentInfo SEQ → [0] EXPLICIT → SignedData SEQ):
	//   decoded.children[1].children[0] is the SignedData SEQUENCE
	//   .children[0]                        = version INTEGER (1)
	//   .children[1]                        = digestAlgorithms SET (1 entry)
	//   .children[1].children[0].children[0] = digestAlgorithm OID
	config := `
locals {
  key     = provider::burnham::ed25519_key_from_seed("ed25519-cms-e2e")
  cert    = provider::burnham::x509_self_sign(local.key, "burnham.ed25519", "serial-15-bytes", "2001-01-01T00:00:00Z", "2099-01-01T00:00:00Z")
  signed  = provider::burnham::pkcs7_sign("hello, ed25519", local.key, local.cert)
  decoded = provider::burnham::asn1_decode(local.signed)
  sd      = local.decoded.children[1].children[0]
}
output "outer_type"          { value = local.decoded.type }
output "signed_data_version" { value = local.sd.children[0].value }
output "digest_oid"          { value = local.sd.children[1].children[0].children[0].value }
`
	runOutputTest(t, config,
		statecheck.ExpectKnownOutputValue("outer_type", knownvalue.StringExact("SEQUENCE")),
		statecheck.ExpectKnownOutputValue("signed_data_version", knownvalue.StringExact("1")),
		// id-sha512 = 2.16.840.1.101.3.4.2.3 per RFC 8419 §3.
		statecheck.ExpectKnownOutputValue("digest_oid", knownvalue.StringExact("2.16.840.1.101.3.4.2.3")),
	)
}

func TestAcc_PKCS7Sign_Ed25519Deterministic(t *testing.T) {
	config := `
locals {
  key  = provider::burnham::ed25519_key_from_seed("det")
  cert = provider::burnham::x509_self_sign(local.key, "cn", "serial-15-bytes", "2001-01-01T00:00:00Z", "2099-01-01T00:00:00Z")
  a    = provider::burnham::pkcs7_sign("payload", local.key, local.cert)
  b    = provider::burnham::pkcs7_sign("payload", local.key, local.cert)
}
output "test" { value = local.a == local.b }
`
	runOutputTest(t, config, statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)))
}

func TestAcc_PKCS7Sign_Ed25519RejectsMismatchedCert(t *testing.T) {
	// Cert from ECDSA-P256 identity but signing key is Ed25519 → key/cert public-key types diverge, mismatch check should fire.
	config := `
locals {
  ecdsa_key  = provider::burnham::ecdsa_p256_key_from_seed("seed")
  ecdsa_cert = provider::burnham::x509_self_sign(local.ecdsa_key, "cn", "serial-15-bytes", "2001-01-01T00:00:00Z", "2099-01-01T00:00:00Z")
  ed_key     = provider::burnham::ed25519_key_from_seed("seed")
}
output "test" { value = provider::burnham::pkcs7_sign("payload", local.ed_key, local.ecdsa_cert) }
`
	runErrorTest(t, config, regexp.MustCompile(`(?is)cert_pem\s+public\s+key\s+does\s+not\s+match`))
}
