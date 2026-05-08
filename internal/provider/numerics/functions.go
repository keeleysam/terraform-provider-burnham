package numerics

import "github.com/hashicorp/terraform-plugin-framework/function"

// Functions returns the numerics provider-defined functions registered by terraform-burnham. RFC 3091 (Pi Digit Generation Protocol) plus statistics and small math helpers that belong in any "numerics" toolbox.
func Functions() []func() function.Function {
	return []func() function.Function{
		// RFC 3091 — Pi Digit Generation Protocol
		NewPiDigitFunction,
		NewPiDigitsFunction,
		NewPiApproximateDigitFunction,
		NewPiApproximateDigitsFunction,

		// Statistics
		NewMeanFunction,
		NewMedianFunction,
		NewPercentileFunction,
		NewVarianceFunction,
		NewStddevFunction,
		NewModeFunction,

		// Math helpers
		NewModFloorFunction,
		NewClampFunction,
	}
}
