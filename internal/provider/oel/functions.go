// Package oel provides Terraform provider functions to build, decode, validate, format, and evaluate Okta Expression Language (OEL) expressions from HCL data.
//
// OEL is the expression language behind Okta group rules, profile mappings, sign-on policy conditions, and Okta Identity Governance policies. It is a documented subset of Spring Expression Language (SpEL).
//
// The functions are backed by github.com/keeleysam/okta-expression-parser (currently a fork that extends the parser to the full documented grammar, pending an upstream contribution).
package oel

import "github.com/hashicorp/terraform-plugin-framework/function"

// Functions returns the Okta Expression Language provider functions registered by terraform-burnham: oelencode (build an OEL string from HCL data), oeldecode (parse an OEL string back into the data tree), oelvalidate (report whether a string is valid OEL), oelformat (canonicalize an OEL string), and oelevaluate (evaluate an OEL expression against a sample context).
func Functions() []func() function.Function {
	return []func() function.Function{
		NewOELEncodeFunction,
		NewOELDecodeFunction,
		NewOELValidateFunction,
		NewOELFormatFunction,
		NewOELEvaluateFunction,
	}
}
