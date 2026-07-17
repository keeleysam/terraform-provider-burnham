// Package color provides pure, deterministic color-manipulation provider functions: parse and reformat CSS colors, compute WCAG contrast, pick legible text colors, generate distinct color sets and ramps, blend, and adjust channels. All math is done in the perceptually-uniform OKLab / OKLCh space and serialized back to sRGB, mirroring CSS Color 4 semantics. Backed by github.com/lucasb-eyer/go-colorful and github.com/mazznoer/csscolorparser, both pure-Go. No randomness: the same inputs always produce the same output, so plan equals apply.
package color

import "github.com/hashicorp/terraform-plugin-framework/function"

// Functions returns the color provider-defined functions registered by terraform-burnham: color_convert (parse any CSS color and reformat it), color_contrast_ratio (WCAG 2.x ratio), color_readable_text (pick legible text for a background), color_distinct (N maximally-distinct colors), color_mix (blend two colors), color_ramp (interpolate N colors across stops), and color_adjust (nudge OKLCh channels).
func Functions() []func() function.Function {
	return []func() function.Function{
		NewColorConvertFunction,
		NewColorContrastRatioFunction,
		NewColorReadableTextFunction,
		NewColorDistinctFunction,
		NewColorMixFunction,
		NewColorRampFunction,
		NewColorAdjustFunction,
	}
}
