// Package image provides pure, deterministic image provider-defined functions:
// svg_render rasterizes an SVG to a PNG via resvg (run as WebAssembly under
// wazero), CGO-free and byte-identical across architectures.
package image

import "github.com/hashicorp/terraform-plugin-framework/function"

// Functions returns the image provider-defined functions registered by
// terraform-burnham: svg_render (rasterize an SVG to a base64 PNG).
func Functions() []func() function.Function {
	return []func() function.Function{
		NewSVGRenderFunction,
	}
}
