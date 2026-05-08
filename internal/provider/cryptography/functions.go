// Package cryptography provides cryptographic / certificate-handling provider-defined functions: HMAC, HKDF, PEM block decoding, X.509 certificate inspection and fingerprinting, PKCS #10 CSR inspection, and generic ASN.1 BER/DER tree decoding.
package cryptography

import "github.com/hashicorp/terraform-plugin-framework/function"

// Functions returns the cryptography provider-defined functions registered by terraform-burnham.
func Functions() []func() function.Function {
	return []func() function.Function{
		NewHMACFunction,
		NewHKDFFunction,
		NewPEMDecodeFunction,
		NewX509InspectFunction,
		NewX509FingerprintFunction,
		NewCSRInspectFunction,
		NewASN1DecodeFunction,
	}
}
