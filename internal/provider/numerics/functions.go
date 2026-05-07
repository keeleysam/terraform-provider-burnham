package numerics

import "github.com/hashicorp/terraform-plugin-framework/function"

// Functions returns the numerics provider-defined functions registered by
// terraform-burnham. Currently this is RFC 3091 (Pi Digit Generation Protocol);
// future numerical / standards-based functions go in this same family.
func Functions() []func() function.Function {
	return []func() function.Function{
		// RFC 3091 — Pi Digit Generation Protocol
		NewPiDigitFunction,
		NewPiDigitsFunction,
		NewPiApproximateDigitFunction,
		NewPiApproximateDigitsFunction,
	}
}
