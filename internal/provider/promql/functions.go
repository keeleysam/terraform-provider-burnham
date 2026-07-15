package promql

import "github.com/hashicorp/terraform-plugin-framework/function"

// Functions returns the PromQL provider functions registered by terraform-burnham: promqlencode (build a query from HCL data), promqlvalidate (report whether a query is valid), and promqlformat (canonicalize a query).
func Functions() []func() function.Function {
	return []func() function.Function{
		NewPromQLEncodeFunction,
		NewPromQLValidateFunction,
		NewPromQLFormatFunction,
	}
}
