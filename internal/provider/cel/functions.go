// Package cel provides Terraform provider functions to build, decode, validate, format, and evaluate CEL (Common Expression Language) expressions from HCL data.
package cel

import "github.com/hashicorp/terraform-plugin-framework/function"

// Functions returns the CEL provider functions registered by terraform-burnham: celencode (build a CEL string from HCL data), celdecode (parse CEL back into the data tree), celvalidate (report whether a CEL string is valid), celformat (canonicalize/pretty-print a CEL string), and celevaluate (evaluate a standard CEL expression).
func Functions() []func() function.Function {
	return []func() function.Function{
		NewCELEncodeFunction,
		NewCELDecodeFunction,
		NewCELValidateFunction,
		NewCELFormatFunction,
		NewCELEvaluateFunction,
	}
}
