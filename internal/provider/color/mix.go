package color

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/keeleysam/terraform-burnham/internal/provider/optionsutil"
	colorful "github.com/lucasb-eyer/go-colorful"
)

// blendSpaces are the interpolation spaces color_mix and color_ramp accept. OKLCh is the default because it interpolates hue and lightness perceptually, avoiding the muddy midpoints of naive sRGB.
var blendSpaces = map[string]func(a, b colorful.Color, t float64) colorful.Color{
	"oklch": func(a, b colorful.Color, t float64) colorful.Color { return a.BlendOkLch(b, t) },
	"oklab": func(a, b colorful.Color, t float64) colorful.Color { return a.BlendOkLab(b, t) },
	"rgb":   func(a, b colorful.Color, t float64) colorful.Color { return a.BlendRgb(b, t) },
	"hsv":   func(a, b colorful.Color, t float64) colorful.Color { return a.BlendHsv(b, t) },
	"lab":   func(a, b colorful.Color, t float64) colorful.Color { return a.BlendLab(b, t) },
	"hcl":   func(a, b colorful.Color, t float64) colorful.Color { return a.BlendHcl(b, t) },
}

func blendFunc(space string) (func(a, b colorful.Color, t float64) colorful.Color, error) {
	fn, ok := blendSpaces[strings.ToLower(strings.TrimSpace(space))]
	if !ok {
		return nil, fmt.Errorf("unknown interpolation space %q; supported: oklch, oklab, rgb, hsv, lab, hcl", space)
	}
	return fn, nil
}

// mixColors blends a and b at position t in [0,1] (0 = a, 1 = b) in the given space, interpolating alpha linearly. Output is hex.
func mixColors(a, b string, t float64, space string) (string, error) {
	fn, err := blendFunc(space)
	if err != nil {
		return "", err
	}
	ca, aa, err := parseColor(a)
	if err != nil {
		return "", err
	}
	cb, ab, err := parseColor(b)
	if err != nil {
		return "", err
	}
	t = clamp01(t)
	mixed := fn(ca, cb, t)
	alpha := aa*(1-t) + ab*t
	return hexOut(mixed, alpha, true, false), nil
}

//go:embed descriptions/color_mix.md
var colorMixDescription string

var _ function.Function = (*ColorMixFunction)(nil)

type ColorMixFunction struct{}

func NewColorMixFunction() function.Function { return &ColorMixFunction{} }

func (f *ColorMixFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "color_mix"
}

func (f *ColorMixFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Blend two colors at a given ratio (OKLCh by default)",
		MarkdownDescription: colorMixDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "a", Description: "The first color (returned when amount is 0)."},
			function.StringParameter{Name: "b", Description: "The second color (returned when amount is 1)."},
			function.Float64Parameter{Name: "amount", Description: "Blend position in [0, 1]; values outside are clamped."},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: "An optional object. Key: `space` (interpolation space, default `oklch`; one of `oklch`, `oklab`, `rgb`, `hsv`, `lab`, `hcl`).",
		},
		Return: function.StringReturn{},
	}
}

func (f *ColorMixFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var a, b string
	var amount float64
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &a, &b, &amount, &optsArgs))
	if resp.Error != nil {
		return
	}
	if unknownStringOptionResult(ctx, resp, optsArgs) {
		return
	}
	space, ferr := spaceOption(optsArgs, 3)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	out, err := mixColors(a, b, amount, space)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}

// spaceOption parses the optional { space = "..." } object, defaulting to oklch. argIndex tags errors at the options parameter position.
func spaceOption(opts []types.Dynamic, argIndex int) (string, *function.FuncError) {
	attrs, ferr := optionsutil.SingleOptionsObject(opts, `{ space = "oklch" }`)
	if ferr != nil {
		return "oklch", ferr
	}
	if attrs == nil {
		return "oklch", nil
	}
	for k, v := range attrs {
		if k != "space" {
			return "oklch", function.NewArgumentFuncError(int64(argIndex), fmt.Sprintf("unknown option key %q; the only key is space", k))
		}
		s, ok := v.(basetypes.StringValue)
		if !ok || s.IsNull() {
			return "oklch", function.NewArgumentFuncError(int64(argIndex), "options.space must be a string")
		}
		return s.ValueString(), nil
	}
	return "oklch", nil
}
