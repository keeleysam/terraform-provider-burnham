// Package identifiers provides identifier-generation provider-defined functions: deterministic UUIDs (v5, v7, plus uuid_inspect), Nano ID, and petname. Every function is pure and seeded — same inputs always produce the same output, so plans don't churn on re-apply.
package identifiers

import "github.com/hashicorp/terraform-plugin-framework/function"

// Functions returns the identifier provider-defined functions registered by terraform-burnham.
func Functions() []func() function.Function {
	return []func() function.Function{
		NewUUIDv5Function,
		NewUUIDv7Function,
		NewUUIDInspectFunction,
		NewNanoidFunction,
		NewPetnameFunction,
	}
}
