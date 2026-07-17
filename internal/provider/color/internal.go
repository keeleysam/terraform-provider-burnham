package color

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	colorful "github.com/lucasb-eyer/go-colorful"
	"github.com/mazznoer/csscolorparser"
)

// parseColor parses any CSS Color 4 notation (hex, rgb()/rgba(), hsl(), hwb(), lab()/lch(), oklab()/oklch(), and named colors) into a go-colorful sRGB color plus its alpha in [0,1]. csscolorparser does the parsing; we keep chroma math in go-colorful.
func parseColor(s string) (colorful.Color, float64, error) {
	cc, err := csscolorparser.Parse(strings.TrimSpace(s))
	if err != nil {
		return colorful.Color{}, 0, fmt.Errorf("could not parse color %q: it must be a CSS color such as \"#1e3a8a\", \"rgb(30 58 138)\", \"oklch(0.3 0.13 265)\", or \"rebeccapurple\"", s)
	}
	return colorful.Color{R: cc.R, G: cc.G, B: cc.B}, cc.A, nil
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// hexOut serializes c to a hex string, appending the alpha byte only when alpha < 1 (so opaque colors stay #rrggbb, the common case). hash and upper control the leading "#" and letter case. The color is gamut-clamped first: OKLCh math can land just outside sRGB.
func hexOut(c colorful.Color, alpha float64, hash, upper bool) string {
	r, g, b := c.Clamped().RGB255()
	var s string
	if alpha >= 1 {
		s = fmt.Sprintf("%02x%02x%02x", r, g, b)
	} else {
		a := uint8(math.Round(clamp01(alpha) * 255))
		s = fmt.Sprintf("%02x%02x%02x%02x", r, g, b, a)
	}
	if upper {
		s = strings.ToUpper(s)
	}
	if hash {
		return "#" + s
	}
	return s
}

// oklchOf returns c in OKLCh with the hue sanitized: for a near-achromatic color go-colorful reports a NaN hue, which would poison any hue arithmetic, so we report 0 there.
func oklchOf(c colorful.Color) (l, chroma, hue float64) {
	l, chroma, hue = c.OkLch()
	if math.IsNaN(hue) {
		hue = 0
	}
	return l, chroma, hue
}

// ── unknown handling ───────────────────────────────────────────
// Terraform auto-defers a call only when a whole argument is unknown, so a known options object carrying an unknown field still reaches Run. These mirror the helpers other burnham packages use.

func hasUnknown(v attr.Value) bool {
	if v == nil {
		return false
	}
	if v.IsUnknown() {
		return true
	}
	switch val := v.(type) {
	case basetypes.DynamicValue:
		return hasUnknown(val.UnderlyingValue())
	case basetypes.TupleValue:
		return elementsHaveUnknown(val.Elements())
	case basetypes.ListValue:
		return elementsHaveUnknown(val.Elements())
	case basetypes.SetValue:
		return elementsHaveUnknown(val.Elements())
	case basetypes.ObjectValue:
		return attributesHaveUnknown(val.Attributes())
	case basetypes.MapValue:
		return attributesHaveUnknown(val.Elements())
	}
	return false
}

func elementsHaveUnknown(elems []attr.Value) bool {
	for _, e := range elems {
		if hasUnknown(e) {
			return true
		}
	}
	return false
}

func attributesHaveUnknown(attrs map[string]attr.Value) bool {
	for _, a := range attrs {
		if hasUnknown(a) {
			return true
		}
	}
	return false
}

// unknownStringOptionResult sets an unknown string result and returns true when any options object carries an unknown value, so a plan-time value that would otherwise silently use defaults resolves at apply instead.
func unknownStringOptionResult(ctx context.Context, resp *function.RunResponse, opts []types.Dynamic) bool {
	for _, o := range opts {
		if hasUnknown(o) {
			resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.StringUnknown()))
			return true
		}
	}
	return false
}
