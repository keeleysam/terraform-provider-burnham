/*
`pkcs7_sign` — CMS/PKCS#7 sign arbitrary bytes with an ECDSA P-256 identity, deterministically.

Builds a SignedData (RFC 5652) ContentInfo: encapsulated `id-data`, no signed attributes (RFC 5652 §5.3 — `signedAttrs` is OPTIONAL when the content type is `id-data`), embedded signer cert, signature over the raw content using RFC 6979 deterministic ECDSA-with-SHA256. The output is byte-identical across runs for the same inputs.

The no-signed-attributes choice matches the wire format Apple's configuration-profile installer accepts on macOS — and avoids the `signingTime: time.Now()` attribute that would re-introduce non-determinism. CMS signing libraries that always add signed attributes can't produce this shape; we use [`github.com/digitorus/pkcs7`](https://github.com/digitorus/pkcs7)'s `SignWithoutAttr` for it.

Output is base64-encoded DER bytes (binary at the HCL boundary, ASCII-safe in transit). Pair with `local_file.content_base64` to write the signed file to disk.
*/

package cryptography

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"fmt"

	"github.com/digitorus/pkcs7"
	"github.com/hashicorp/terraform-plugin-framework/function"
)

// pkcs7DataMaxBytes caps the `data` input to defeat adversarial multi-gigabyte payloads. 16 MiB matches the existing CBOR / MessagePack / VDF / KDL family caps; mobileconfigs and similar real-world signing inputs sit several orders of magnitude under it.
const pkcs7DataMaxBytes = 16 * 1024 * 1024

var _ function.Function = (*PKCS7SignFunction)(nil)

type PKCS7SignFunction struct{}

func NewPKCS7SignFunction() function.Function { return &PKCS7SignFunction{} }

func (f *PKCS7SignFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "pkcs7_sign"
}

func (f *PKCS7SignFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "CMS / PKCS#7 sign bytes with an ECDSA P-256 identity (deterministic; RFC 5652 §5.3 no-signed-attrs shape)",
		MarkdownDescription: fmt.Sprintf("Produces a CMS SignedData ContentInfo (RFC 5652) carrying `data` as its encapsulated content. The signer cert is embedded; the signature is RFC 6979 deterministic ECDSA-with-SHA256 over the raw content; no signed attributes are included (RFC 5652 §5.3 permits omitting `signedAttrs` when the encapsulated content type is `id-data`).\n\nDeterministic by construction: identical `(data, private_key_pem, cert_pem)` always yields the same DER bytes.\n\nOutput is base64-encoded DER. Decode with `base64decode(...)` or feed straight into `local_file.content_base64`.\n\n```\nresource \"local_file\" \"signed_profile\" {\n  filename       = \"signed/profile.mobileconfig\"\n  content_base64 = provider::burnham::pkcs7_sign(\n    file(\"profile.mobileconfig\"),\n    local.signer_key_pem,\n    local.signer_cert_pem,\n  )\n}\n```\n\nKey/cert constraints:\n\n- `private_key_pem`: ECDSA P-256, PKCS#8 or SEC1.\n- `cert_pem`: X.509 cert whose public key matches the private key (checked at call time — a mismatch is rejected rather than producing an unverifiable signature). No chain validation is performed; this is the on-the-wire signing primitive, not a PKI workflow.\n- `data` must be 1 byte to %d bytes (%d MiB).\n\nCompliance posture: the emitted SignedData has `version: 1` and `SignerInfo.version: 1` per RFC 5652 §5.1 / §5.3 (encapsulated content is `id-data`, signer identified by `issuerAndSerialNumber`, no version-3-certificate or OtherCertificateFormat children).\n\nApple configuration-profile signing uses exactly this shape; Jamf passes signed profiles through unchanged. For other use cases that need the more typical CMS shape *with* signed attributes (and the resulting `signingTime` non-determinism), use a different library — this function is intentionally the no-signed-attrs flavour.\n\n%s", pkcs7DataMaxBytes, pkcs7DataMaxBytes/(1024*1024), hclByteHandlingGotcha),
		Parameters: []function.Parameter{
			function.StringParameter{Name: "data", Description: fmt.Sprintf("Content to sign (raw bytes). Must not be empty and must not exceed %d bytes (%d MiB).", pkcs7DataMaxBytes, pkcs7DataMaxBytes/(1024*1024))},
			function.StringParameter{Name: "private_key_pem", Description: "PEM-encoded ECDSA P-256 private key (`PRIVATE KEY` PKCS#8 or `EC PRIVATE KEY` SEC1)."},
			function.StringParameter{Name: "cert_pem", Description: "PEM-encoded X.509 certificate whose public key matches the private key."},
		},
		Return: function.StringReturn{},
	}
}

func (f *PKCS7SignFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var data, keyPEM, certPEM string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &data, &keyPEM, &certPEM))
	if resp.Error != nil {
		return
	}
	if len(data) == 0 {
		resp.Error = function.NewArgumentFuncError(0, "data must not be empty")
		return
	}
	if len(data) > pkcs7DataMaxBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("data exceeds maximum length: %d bytes; got %d", pkcs7DataMaxBytes, len(data)))
		return
	}

	key, err := parseECDSAP256PrivateKey(keyPEM)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(1, err.Error())
		return
	}
	certDER, err := firstPEMBlockBytes(certPEM, "CERTIFICATE")
	if err != nil {
		resp.Error = function.NewArgumentFuncError(2, "cert_pem: "+err.Error())
		return
	}
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(2, "cert_pem: parse: "+err.Error())
		return
	}
	// Without this check, mismatched key/cert silently produces a CMS that no verifier can validate — a class of late-stage failure (rejected at install time) that's cheap to catch here.
	if !key.PublicKey.Equal(cert.PublicKey) {
		resp.Error = function.NewArgumentFuncError(2, "cert_pem public key does not match private_key_pem")
		return
	}

	sd, err := pkcs7.NewSignedData([]byte(data))
	if err != nil {
		resp.Error = function.NewFuncError("CMS init failed: " + err.Error())
		return
	}
	sd.SetDigestAlgorithm(pkcs7.OIDDigestAlgorithmSHA256)
	if err := sd.SignWithoutAttr(cert, &detECDSASigner{priv: key}, pkcs7.SignerInfoConfig{}); err != nil {
		resp.Error = function.NewFuncError("CMS signing failed: " + err.Error())
		return
	}
	der, err := sd.Finish()
	if err != nil {
		resp.Error = function.NewFuncError("CMS finalize failed: " + err.Error())
		return
	}
	out := base64.StdEncoding.EncodeToString(der)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}
