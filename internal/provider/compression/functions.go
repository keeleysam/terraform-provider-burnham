// Package compression provides compression-oriented provider-defined functions that emit base64-encoded compressed payloads: base64zopfli (a drop-in, RFC 1952 gzip replacement for the built-in base64gzip, using Zopfli's iterative DEFLATE encoder for a tighter result) and base64brotli (Brotli, RFC 7932, for a larger win at the cost of a brotli decompressor on the consuming side). Both are pure and deterministic — identical input and options always produce byte-identical output, so Terraform plans stay stable.
package compression

import "github.com/hashicorp/terraform-plugin-framework/function"

// Functions returns the compression provider-defined functions registered by terraform-burnham.
func Functions() []func() function.Function {
	return []func() function.Function{
		NewBase64ZopfliFunction,
		NewBase64BrotliFunction,
	}
}
