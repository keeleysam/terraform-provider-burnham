/*
`x509_self_sign` — build a self-signed X.509 certificate from a PEM-encoded private key, deterministically.

Signs with RFC 6979 deterministic ECDSA via `detECDSASigner`, so identical inputs produce byte-identical certs. Pair with [`ecdsa_p256_key_from_seed`](#function-ecdsa_p256_key_from_seed) for a fully deterministic identity, or pass a long-lived externally-managed key to get a stable signing identity that doesn't churn in Terraform state.

Currently P-256 only; the signer wrapper is ECDSA-specific. RSA and Ed25519 paths would each need their own deterministic wrappers (RSA is naturally deterministic with PKCS#1 v1.5; Ed25519 is naturally deterministic by spec).
*/

package cryptography

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

var _ function.Function = (*X509SelfSignFunction)(nil)

type X509SelfSignFunction struct{}

func NewX509SelfSignFunction() function.Function { return &X509SelfSignFunction{} }

func (f *X509SelfSignFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "x509_self_sign"
}

func (f *X509SelfSignFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Build a deterministic self-signed X.509 cert from a PEM private key",
		MarkdownDescription: fmt.Sprintf("Constructs a self-signed X.509 v3 certificate signed with RFC 6979 deterministic ECDSA. Given the same `private_key_pem` and the same parameters, the output is byte-identical across runs.\n\nPaired with [`ecdsa_p256_key_from_seed`](#function-ecdsa_p256_key_from_seed), the full chain from input seed → key → cert is deterministic — no random state involved at any step.\n\nFields produced:\n\n- **Version**: 3.\n- **Serial Number**: derived from `serial` (raw bytes; interpreted big-endian, leading-byte high bit cleared so the DER-encoded length stays predictable). 8–20 bytes — RFC 5280 §4.1.2.2 caps the encoded length at 20 octets.\n- **Issuer = Subject**: a single Common Name attribute (self-signed).\n- **Validity**: as supplied, RFC 3339.\n- **Basic Constraints**: critical, `CA:FALSE`.\n- **Signature Algorithm**: `ecdsa-with-SHA256`.\n\nOnly ECDSA P-256 keys are accepted; other key types return an error. PEM input must contain one of `PRIVATE KEY` (PKCS#8) or `EC PRIVATE KEY` (SEC1) blocks.\n\n```\nprovider::burnham::x509_self_sign(\n  provider::burnham::ecdsa_p256_key_from_seed(sha512(file(\"input.bin\"))),\n  \"signer.example\",\n  provider::burnham::hkdf(\"sha256\", sha512(file(\"input.bin\")), \"\", \"serial\", 10),\n  \"2001-01-01T00:00:00Z\",\n  \"2099-01-01T00:00:00Z\",\n)\n→ \"-----BEGIN CERTIFICATE-----\\nMIIB…\\n-----END CERTIFICATE-----\\n\"\n```\n\n%s", hclByteHandlingGotcha),
		Parameters: []function.Parameter{
			function.StringParameter{Name: "private_key_pem", Description: "PEM-encoded ECDSA P-256 private key (`PRIVATE KEY` PKCS#8 or `EC PRIVATE KEY` SEC1)."},
			function.StringParameter{Name: "common_name", Description: "Subject / Issuer Common Name (also used as the DN). 1–64 characters (counted as Unicode code points per RFC 5280 §A.1 `ub-common-name-length`, not bytes — a 30-CJK-character CN passes even though it's 90 UTF-8 bytes)."},
			function.StringParameter{Name: "serial", Description: "Serial number source bytes (raw bytes). Interpreted as a big-endian unsigned integer; the leading-byte high bit is cleared to keep the DER-encoded length stable. RFC 5280 §4.1.2.2 caps the DER-encoded INTEGER at 20 octets — pass at least 8 bytes for uniqueness, at most 20 for compliance. **Pairing with `hkdf`:** because `hkdf` returns hex (so 20 raw bytes here equals 20 hex characters = 10 actual entropy bytes from HKDF), request `hkdf(\"sha256\", seed, \"\", info, 10)` for a serial that lands right at the 20-octet cap."},
			function.StringParameter{Name: "not_before", Description: "Validity start, RFC 3339 (e.g. `2025-01-01T00:00:00Z`)."},
			function.StringParameter{Name: "not_after", Description: "Validity end, RFC 3339. Must be strictly after `not_before`."},
		},
		Return: function.StringReturn{},
	}
}

func (f *X509SelfSignFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var keyPEM, commonName, serialBytes, notBeforeStr, notAfterStr string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &keyPEM, &commonName, &serialBytes, &notBeforeStr, &notAfterStr))
	if resp.Error != nil {
		return
	}

	// RFC 5280 §A.1 sets ub-common-name-length := 64 — that's a 64-*character* cap (ASN.1 CHARACTER STRING), not 64 bytes. Multi-byte UTF-8 input would be rejected by a byte-based check even when RFC-legal, so we count runes.
	if runes := utf8.RuneCountInString(commonName); runes < 1 || runes > 64 {
		resp.Error = function.NewArgumentFuncError(1, fmt.Sprintf("common_name must be 1–64 characters; got %d", runes))
		return
	}
	if len(serialBytes) == 0 {
		resp.Error = function.NewArgumentFuncError(2, "serial must not be empty")
		return
	}
	notBefore, err := time.Parse(time.RFC3339, notBeforeStr)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(3, "not_before is not RFC 3339: "+err.Error())
		return
	}
	notAfter, err := time.Parse(time.RFC3339, notAfterStr)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(4, "not_after is not RFC 3339: "+err.Error())
		return
	}
	if !notAfter.After(notBefore) {
		resp.Error = function.NewArgumentFuncError(4, "not_after must be strictly after not_before")
		return
	}

	key, err := parseECDSAP256PrivateKey(keyPEM)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	der, err := selfSignECDSAP256(key, commonName, []byte(serialBytes), notBefore, notAfter)
	if err != nil {
		resp.Error = function.NewFuncError("certificate signing failed: " + err.Error())
		return
	}
	out := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}

// selfSignECDSAP256 builds and signs the cert. Internal because x509_self_sign and pkcs7_sign internal tests want to construct certs without going through the public function surface.
//
// Compliance posture: produces a v3 X.509 certificate that satisfies every MUST-level requirement of RFC 5280 we exercise — v3 structure (§4.1), positive serial ≤ 20 octets (§4.1.2.2), UTCTime/GeneralizedTime split at year 2050 (§4.1.2.5) inherited from `crypto/x509.CreateCertificate`, BasicConstraints present with cA=FALSE (§4.2.1.9).
//
// Two SHOULD-level extensions are intentionally not set:
//   - SubjectKeyIdentifier (§4.2.1.2) — RFC 5280 says "for end entity certificates, subject key identifiers SHOULD be derived from the public key." This SHOULD does apply to us and is unfulfilled.
//   - KeyUsage (§4.2.1.3) — RFC 5280 scopes its SHOULD to certs "that contain public keys that are used to validate digital signatures on other public key certificates or CRLs." Our certs sign mobileconfigs and CMS payloads, not other certs/CRLs, so §4.2.1.3 SHOULD doesn't strictly bind us — but `digitalSignature` is the conventional usage bit for a signer cert and its absence is noted here for completeness.
//
// The cert is intended for the on-the-wire signing primitive (mobileconfig signing, CMS signers) rather than a PKI deployment, and the macOS profile installer doesn't require either extension. Callers needing strict PKI deployment should produce the cert outside this function and pass it into `pkcs7_sign` directly.
func selfSignECDSAP256(key *ecdsa.PrivateKey, commonName string, serialBytes []byte, notBefore, notAfter time.Time) ([]byte, error) {
	serial := new(big.Int).SetBytes(serialBytes)
	// Clear the high bit of the leading byte so the DER-encoded INTEGER stays the same length as the input bytes — without this, `crypto/x509` would prepend a 0x00 padding byte (DER's positivity-preservation rule) and the encoded length grows by one. The cert is still valid either way; this just keeps the serial length predictable.
	if serial.Sign() > 0 {
		serial.SetBit(serial, serial.BitLen()-1, 0)
	}
	if serial.Sign() == 0 {
		serial.SetInt64(1)
	}
	// RFC 5280 §4.1.2.2 caps serialNumber at 20 octets when DER-encoded. With the high bit cleared above, the DER length equals the byte length of the magnitude, so we can compare directly.
	if magBytes := (serial.BitLen() + 7) / 8; magBytes > 20 {
		return nil, fmt.Errorf("serial would encode to %d octets; RFC 5280 §4.1.2.2 limits this to 20", magBytes)
	}

	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: commonName},
		Issuer:                pkix.Name{CommonName: commonName},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		BasicConstraintsValid: true,
		IsCA:                  false,
		SignatureAlgorithm:    x509.ECDSAWithSHA256,
	}
	return x509.CreateCertificate(nil, template, template, &key.PublicKey, &detECDSASigner{priv: key})
}

// parseECDSAP256PrivateKey accepts either PKCS#8 `PRIVATE KEY` or SEC1 `EC PRIVATE KEY` PEM blocks and returns the contained P-256 ECDSA key. Other curves and other key types are rejected.
func parseECDSAP256PrivateKey(pemStr string) (*ecdsa.PrivateKey, error) {
	der, err := firstPEMBlockBytes(pemStr, "PRIVATE KEY", "EC PRIVATE KEY")
	if err != nil {
		return nil, fmt.Errorf("private_key_pem: %w", err)
	}
	// Try PKCS#8 first (the format we emit from ecdsa_p256_key_from_seed); fall back to SEC1.
	if k, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		ec, ok := k.(*ecdsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private_key_pem: expected ECDSA key, got %T", k)
		}
		if ec.Curve != elliptic.P256() {
			return nil, fmt.Errorf("private_key_pem: expected P-256, got %s", ec.Curve.Params().Name)
		}
		return ec, nil
	}
	ec, err := x509.ParseECPrivateKey(der)
	if err != nil {
		return nil, fmt.Errorf("private_key_pem: not a recognized ECDSA private key (PKCS#8 or SEC1)")
	}
	if ec.Curve != elliptic.P256() {
		return nil, fmt.Errorf("private_key_pem: expected P-256, got %s", ec.Curve.Params().Name)
	}
	return ec, nil
}
