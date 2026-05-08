/*
PKCS #10 CSR inspection.

Most of the certificate-management workflows that benefit from `x509_inspect` also need to look at CSRs at plan time — to assert that the CN matches a managed value, that the SANs cover the right names, that the signature algorithm is what the CA expects. This is the CSR analogue: same structured-output approach, parsing PKCS #10 (RFC 2986) certificate signing requests.
*/

package cryptography

import (
	"context"
	"crypto/x509"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = (*CSRInspectFunction)(nil)

type CSRInspectFunction struct{}

func NewCSRInspectFunction() function.Function { return &CSRInspectFunction{} }

func (f *CSRInspectFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "csr_inspect"
}

// csrInspectAttrs is the fixed-shape object returned by csr_inspect. Note: CSRs do not carry a serial number, validity window, key usage, ExtKeyUsage, or BasicConstraints CA flag — those are set by the issuing CA when the request is approved. So the schema is a strict subset of x509_inspect's.
var csrInspectAttrs = map[string]attr.Type{
	"subject":              types.StringType,
	"signature_algorithm":  types.StringType,
	"public_key_algorithm": types.StringType,
	"dns_names":            types.ListType{ElemType: types.StringType},
	"email_addresses":      types.ListType{ElemType: types.StringType},
	"ip_addresses":         types.ListType{ElemType: types.StringType},
	"uris":                 types.ListType{ElemType: types.StringType},
}

func (f *CSRInspectFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Decode a PEM-encoded PKCS #10 certificate signing request into a structured object",
		MarkdownDescription: "Parses the first `CERTIFICATE REQUEST` block in `pem` and returns a fixed-shape object:\n\n- `subject` — RFC 4514 distinguished-name string from the CSR\n- `signature_algorithm`, `public_key_algorithm` — `SHA256-RSA`, `Ed25519`, etc.\n- `dns_names`, `email_addresses`, `ip_addresses`, `uris` — Subject Alternative Names by category, taken from the CSR's requested-extensions attribute\n\nFields that don't exist on a CSR (serial number, validity window, key usage, BasicConstraints) are not on this object — those are set by the issuing CA when the request is approved.\n\n**This function reads structure, not trust.** It does **not** verify the CSR's self-signature. Treat the result as the *requested* attributes; the issuing CA decides what actually ends up on the certificate.\n\nErrors when the input contains no CERTIFICATE REQUEST block or the request fails to parse.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "pem", Description: "PEM-encoded CSR (or a bundle containing one)."},
		},
		Return: function.ObjectReturn{AttributeTypes: csrInspectAttrs},
	}
}

func (f *CSRInspectFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))
	if resp.Error != nil {
		return
	}
	csr, err := firstCSR(input)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	ipStrings := make([]string, len(csr.IPAddresses))
	for i, ip := range csr.IPAddresses {
		ipStrings[i] = ip.String()
	}
	uriStrings := make([]string, len(csr.URIs))
	for i, u := range csr.URIs {
		uriStrings[i] = u.String()
	}

	out, diags := types.ObjectValue(csrInspectAttrs, map[string]attr.Value{
		"subject":              types.StringValue(csr.Subject.String()),
		"signature_algorithm":  types.StringValue(csr.SignatureAlgorithm.String()),
		"public_key_algorithm": types.StringValue(csr.PublicKeyAlgorithm.String()),
		"dns_names":            mustStringList(csr.DNSNames),
		"email_addresses":      mustStringList(csr.EmailAddresses),
		"ip_addresses":         mustStringList(ipStrings),
		"uris":                 mustStringList(uriStrings),
	})
	if diags.HasError() {
		resp.Error = function.NewFuncError("building inspect object: " + diagsToString(diags))
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}

// firstCSR walks the PEM input for a `CERTIFICATE REQUEST` (or pre-RFC 7468 `NEW CERTIFICATE REQUEST`) block and returns the parsed CSR.
func firstCSR(input string) (*x509.CertificateRequest, error) {
	der, err := firstPEMBlockBytes(input, "CERTIFICATE REQUEST", "NEW CERTIFICATE REQUEST")
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificateRequest(der)
}
