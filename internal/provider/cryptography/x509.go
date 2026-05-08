/*
X.509 certificate inspection and fingerprinting.

`x509_inspect` parses a single PEM-encoded `CERTIFICATE` block and returns a fixed-shape object summarising the fields people actually look at: subject, issuer, validity window, SANs, key/extended-key usage. Concretely useful at plan time for asserting on certificate properties (expiry comparisons, SAN coverage checks) without dropping into a script.

`x509_fingerprint` computes a fingerprint of the DER bytes — the same value `openssl x509 -fingerprint -sha256 …` prints, minus the colon separators.

Both functions accept either a raw PEM-encoded certificate or a PEM bundle whose first `CERTIFICATE` block they use; non-CERTIFICATE blocks are skipped so a `tls_*.crt` containing intermediates works without preprocessing.
*/

package cryptography

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// firstCertificate walks the PEM input for a `CERTIFICATE` block and returns the parsed certificate. Other block types are skipped silently so a mixed bundle still works.
func firstCertificate(input string) (*x509.Certificate, error) {
	der, err := firstPEMBlockBytes(input, "CERTIFICATE")
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(der)
}

// ──────────────────────────────────────────────────────────────────────
// x509_inspect
// ──────────────────────────────────────────────────────────────────────

var _ function.Function = (*X509InspectFunction)(nil)

type X509InspectFunction struct{}

func NewX509InspectFunction() function.Function { return &X509InspectFunction{} }

func (f *X509InspectFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "x509_inspect"
}

// x509InspectAttrs is the fixed-shape object returned by x509_inspect.
var x509InspectAttrs = map[string]attr.Type{
	"subject":              types.StringType,
	"issuer":               types.StringType,
	"serial_number":        types.StringType,
	"not_before":           types.StringType,
	"not_after":            types.StringType,
	"signature_algorithm":  types.StringType,
	"public_key_algorithm": types.StringType,
	"is_ca":                types.BoolType,
	"key_usage":            types.ListType{ElemType: types.StringType},
	"ext_key_usage":        types.ListType{ElemType: types.StringType},
	"dns_names":            types.ListType{ElemType: types.StringType},
	"email_addresses":      types.ListType{ElemType: types.StringType},
	"ip_addresses":         types.ListType{ElemType: types.StringType},
	"uris":                 types.ListType{ElemType: types.StringType},
}

func (f *X509InspectFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Decode a PEM-encoded X.509 certificate into a structured object",
		MarkdownDescription: "Parses the first `CERTIFICATE` block in `pem` and returns a fixed-shape object:\n\n- `subject`, `issuer` — RFC 4514 distinguished-name strings\n- `serial_number` — decimal\n- `not_before`, `not_after` — RFC 3339 timestamps\n- `signature_algorithm`, `public_key_algorithm` — `SHA256-RSA`, `Ed25519`, etc.\n- `is_ca` — bool, true when BasicConstraints CA flag is set\n- `key_usage` — list of name strings drawn from RFC 5280 KeyUsage\n- `ext_key_usage` — list of name strings drawn from RFC 5280 ExtendedKeyUsage\n- `dns_names`, `email_addresses`, `ip_addresses`, `uris` — Subject Alternative Names by category\n\nNon-CERTIFICATE PEM blocks (e.g. private keys, CSRs) before the cert are skipped, so a mixed `chain.pem` works without preprocessing.\n\n**Bundle order matters.** When `pem` contains multiple `CERTIFICATE` blocks (a fullchain.pem, a CMS signature, etc.) this returns the *first* one, not the leaf. The leaf-first ordering is the convention for fullchain.pem and ACME-issued bundles, but a reordered or intermediate-first bundle silently inspects a non-leaf certificate. If your input could be reordered, split the bundle upstream and pass only the leaf.\n\n**This function reads structure, not trust.** It does **not** verify the certificate's signature, validity window, or chain to any trusted root — a self-signed, expired, or revoked blob parses just fine. Don't make security decisions on the result without a separate signing or trust-validation step.\n\nErrors when the input contains no CERTIFICATE block or the certificate fails to parse.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "pem", Description: "PEM-encoded certificate (or a bundle containing one)."},
		},
		Return: function.ObjectReturn{AttributeTypes: x509InspectAttrs},
	}
}

func (f *X509InspectFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))
	if resp.Error != nil {
		return
	}
	cert, err := firstCertificate(input)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	ipStrings := make([]string, len(cert.IPAddresses))
	for i, ip := range cert.IPAddresses {
		ipStrings[i] = ip.String()
	}
	uriStrings := make([]string, len(cert.URIs))
	for i, u := range cert.URIs {
		uriStrings[i] = u.String()
	}

	dnsNames := mustStringList(cert.DNSNames)
	emails := mustStringList(cert.EmailAddresses)
	ips := mustStringList(ipStrings)
	uris := mustStringList(uriStrings)
	keyUsage := mustStringList(keyUsageNames(cert.KeyUsage))
	extKeyUsage := mustStringList(extKeyUsageNames(cert.ExtKeyUsage))

	out, diags := types.ObjectValue(x509InspectAttrs, map[string]attr.Value{
		"subject":              types.StringValue(cert.Subject.String()),
		"issuer":               types.StringValue(cert.Issuer.String()),
		"serial_number":        types.StringValue(cert.SerialNumber.String()),
		"not_before":           types.StringValue(cert.NotBefore.UTC().Format(time.RFC3339)),
		"not_after":            types.StringValue(cert.NotAfter.UTC().Format(time.RFC3339)),
		"signature_algorithm":  types.StringValue(cert.SignatureAlgorithm.String()),
		"public_key_algorithm": types.StringValue(cert.PublicKeyAlgorithm.String()),
		"is_ca":                types.BoolValue(cert.IsCA),
		"key_usage":            keyUsage,
		"ext_key_usage":        extKeyUsage,
		"dns_names":            dnsNames,
		"email_addresses":      emails,
		"ip_addresses":         ips,
		"uris":                 uris,
	})
	if diags.HasError() {
		resp.Error = function.NewFuncError("building inspect object: " + diagsToString(diags))
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}

// ──────────────────────────────────────────────────────────────────────
// x509_fingerprint
// ──────────────────────────────────────────────────────────────────────

var _ function.Function = (*X509FingerprintFunction)(nil)

type X509FingerprintFunction struct{}

func NewX509FingerprintFunction() function.Function { return &X509FingerprintFunction{} }

func (f *X509FingerprintFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "x509_fingerprint"
}

func (f *X509FingerprintFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Hex fingerprint of an X.509 certificate's DER bytes",
		MarkdownDescription: "Returns the hex-encoded `algorithm` digest of the first `CERTIFICATE` block's DER bytes — the same value `openssl x509 -fingerprint -<algorithm>` produces, minus the colon separators between byte pairs.\n\n`algorithm` is one of `\"sha1\"`, `\"sha256\"`, `\"sha384\"`, `\"sha512\"`. (`sha224` is not commonly used for fingerprints and is omitted.) `sha256` is the standard choice in 2026; `sha1` is supported only for compatibility with older systems.\n\n**Bundle order matters.** Like `x509_inspect`, this hashes the *first* `CERTIFICATE` block in the input, which is the leaf in a conventionally-ordered fullchain.pem but not in an intermediate-first bundle. Pre-split the bundle if you need to fingerprint a specific certificate.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "pem", Description: "PEM-encoded certificate (or a bundle containing one)."},
			function.StringParameter{Name: "algorithm", Description: "Hash algorithm: \"sha1\", \"sha256\", \"sha384\", or \"sha512\"."},
		},
		Return: function.StringReturn{},
	}
}

func (f *X509FingerprintFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input, algorithm string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input, &algorithm))
	if resp.Error != nil {
		return
	}
	cert, err := firstCertificate(input)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}
	var digest []byte
	switch algorithm {
	case "sha1":
		s := sha1.Sum(cert.Raw)
		digest = s[:]
	case "sha256":
		s := sha256.Sum256(cert.Raw)
		digest = s[:]
	case "sha384":
		s := sha512.Sum384(cert.Raw)
		digest = s[:]
	case "sha512":
		s := sha512.Sum512(cert.Raw)
		digest = s[:]
	default:
		resp.Error = function.NewArgumentFuncError(1, fmt.Sprintf("algorithm must be sha1, sha256, sha384, or sha512; received %q", algorithm))
		return
	}
	out := hex.EncodeToString(digest)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}
