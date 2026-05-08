// Package geographic provides geocoding provider-defined functions: geohash and Open Location Code (Plus codes) — encode/decode for both.
package geographic

import "github.com/hashicorp/terraform-plugin-framework/function"

// Functions returns the geographic provider-defined functions registered by terraform-burnham.
func Functions() []func() function.Function {
	return []func() function.Function{
		NewGeohashEncodeFunction,
		NewGeohashDecodeFunction,
		NewPluscodeEncodeFunction,
		NewPluscodeDecodeFunction,
	}
}
