// Package encoding provides byte-codec provider-defined functions: hex and base64 encode/decode. These fill gaps in Terraform core — there is no built-in hex decode, and core's `base64encode`/`base64decode` only speak standard, padded base64. The encoders take an options object for the RFC 4648 variants; the decoders are deliberately lenient so they accept whatever an encoder on the other side produced.
package encoding

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

// Functions returns the encoding provider-defined functions registered by terraform-burnham.
func Functions() []func() function.Function {
	return []func() function.Function{
		NewHexEncodeFunction,
		NewHexDecodeFunction,
		NewBase64EncodeFunction,
		NewBase64DecodeFunction,
	}
}

// stripASCIIWhitespace removes spaces, tabs, CR and LF — the leniency every
// decoder in this package shares so wrapped/pretty-printed input round-trips.
func stripASCIIWhitespace(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\r', '\n':
			return -1
		}
		return r
	}, s)
}
