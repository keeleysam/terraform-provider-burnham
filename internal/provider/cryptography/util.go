// Internal helpers shared across the cryptography family.

package cryptography

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// hclByteHandlingGotcha is the canonical explanation of how HCL string literals reach byte-oriented crypto primitives. Both `hmac` and `hkdf` (and any future helper that takes raw-byte arguments) embed this verbatim into their MarkdownDescription so the warning stays in lockstep across functions.
const hclByteHandlingGotcha = "**Byte handling, gotchas:** the inputs reach the function as the literal UTF-8 bytes of whatever string HCL hands it. HCL string literals only support `\\uNNNN` Unicode escapes — there is no `\\xNN` byte-escape syntax. A value spelled `\"\\u00ff\"` arrives as the two UTF-8 bytes `0xc3 0xbf`, *not* the single byte `0xff`. An OpenSSL-style hex value like `\"00ff\"` is similarly interpreted as four ASCII characters, *not* two raw bytes. For arbitrary-byte inputs (RFC test vectors, hex-encoded keys, anything outside ASCII), encode upstream as base64 in your variable and pass `base64decode(var.x)` to this function. Burnham does not currently ship a `hex_decode` helper."

// diagsToString joins all diag.Diagnostics into a single human-readable string. Used so we can surface framework-internal errors via the function.NewFuncError API which only takes a string.
func diagsToString(d diag.Diagnostics) string {
	if !d.HasError() {
		return ""
	}
	parts := make([]string, 0, d.ErrorsCount())
	for _, e := range d.Errors() {
		parts = append(parts, fmt.Sprintf("%s: %s", e.Summary(), e.Detail()))
	}
	return strings.Join(parts, "; ")
}

// firstPEMBlockBytes walks the PEM-armoured input until it finds a block whose Type matches one of the supplied labels, and returns that block's body bytes. Other block types are skipped silently so a mixed bundle (cert + key + CSR + …) works without the caller pre-filtering. Returns an error when no matching block exists.
//
// **Order matters.** This returns the *first* matching block in input order, not the leaf, the most-recent, or any semantically-distinguished block. For a typical fullchain.pem (leaf, intermediate(s), root) the leaf wins by convention, but a reordered or intermediate-only bundle would silently bind to a non-leaf certificate without any error signal. Callers that need leaf-vs-chain semantics must split the bundle upstream.
func firstPEMBlockBytes(input string, accept ...string) ([]byte, error) {
	rest := []byte(input)
	for {
		block, next := pem.Decode(rest)
		if block == nil {
			return nil, fmt.Errorf("no %s block found in input", strings.Join(accept, " or "))
		}
		for _, t := range accept {
			if block.Type == t {
				return block.Bytes, nil
			}
		}
		rest = next
	}
}

// mustStringList builds a Terraform list-of-string attr.Value from a Go []string. ListValue's diagnostics are non-error for the (StringType, []StringValue) input shape we use; on the rare framework bug we fall back to a null list of the right element type rather than panic.
func mustStringList(xs []string) attr.Value {
	vals := make([]attr.Value, len(xs))
	for i, s := range xs {
		vals[i] = types.StringValue(s)
	}
	v, d := types.ListValue(types.StringType, vals)
	if d.HasError() {
		return types.ListNull(types.StringType)
	}
	return v
}

// keyUsageNames maps an x509.KeyUsage bitmask to a sorted list of human-readable usage names. Names match the strings the IETF/CABForum specs use.
func keyUsageNames(u x509.KeyUsage) []string {
	type bit struct {
		mask x509.KeyUsage
		name string
	}
	bits := []bit{
		{x509.KeyUsageDigitalSignature, "digitalSignature"},
		{x509.KeyUsageContentCommitment, "contentCommitment"},
		{x509.KeyUsageKeyEncipherment, "keyEncipherment"},
		{x509.KeyUsageDataEncipherment, "dataEncipherment"},
		{x509.KeyUsageKeyAgreement, "keyAgreement"},
		{x509.KeyUsageCertSign, "keyCertSign"},
		{x509.KeyUsageCRLSign, "cRLSign"},
		{x509.KeyUsageEncipherOnly, "encipherOnly"},
		{x509.KeyUsageDecipherOnly, "decipherOnly"},
	}
	var out []string
	for _, b := range bits {
		if u&b.mask != 0 {
			out = append(out, b.name)
		}
	}
	return out
}

// extKeyUsageNames maps an x509.ExtKeyUsage to its standard short name.
func extKeyUsageNames(usages []x509.ExtKeyUsage) []string {
	out := make([]string, 0, len(usages))
	for _, u := range usages {
		switch u {
		case x509.ExtKeyUsageAny:
			out = append(out, "any")
		case x509.ExtKeyUsageServerAuth:
			out = append(out, "serverAuth")
		case x509.ExtKeyUsageClientAuth:
			out = append(out, "clientAuth")
		case x509.ExtKeyUsageCodeSigning:
			out = append(out, "codeSigning")
		case x509.ExtKeyUsageEmailProtection:
			out = append(out, "emailProtection")
		case x509.ExtKeyUsageIPSECEndSystem:
			out = append(out, "ipsecEndSystem")
		case x509.ExtKeyUsageIPSECTunnel:
			out = append(out, "ipsecTunnel")
		case x509.ExtKeyUsageIPSECUser:
			out = append(out, "ipsecUser")
		case x509.ExtKeyUsageTimeStamping:
			out = append(out, "timeStamping")
		case x509.ExtKeyUsageOCSPSigning:
			out = append(out, "ocspSigning")
		case x509.ExtKeyUsageMicrosoftServerGatedCrypto:
			out = append(out, "microsoftServerGatedCrypto")
		case x509.ExtKeyUsageNetscapeServerGatedCrypto:
			out = append(out, "netscapeServerGatedCrypto")
		case x509.ExtKeyUsageMicrosoftCommercialCodeSigning:
			out = append(out, "microsoftCommercialCodeSigning")
		case x509.ExtKeyUsageMicrosoftKernelCodeSigning:
			out = append(out, "microsoftKernelCodeSigning")
		default:
			out = append(out, fmt.Sprintf("unknown(%d)", int(u)))
		}
	}
	return out
}
