package color

import (
	"context"
	_ "embed"
	"fmt"
	"math"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/keeleysam/terraform-burnham/internal/provider/optionsutil"
	colorful "github.com/lucasb-eyer/go-colorful"
)

// defaultSchemeAngle is the hue spread, in degrees, used by the analogous and split-complementary schemes. 30 is the conventional choice: wide enough to read as separate colors, narrow enough to stay harmonious.
const defaultSchemeAngle = 30.0

// schemeOffsets returns the hue rotations, in degrees, for a named color scheme. The angle parameter only affects analogous and split-complementary; the fixed schemes ignore it. The first offset is always 0 so the base color leads the palette.
func schemeOffsets(scheme string, angle float64) ([]float64, error) {
	switch strings.ToLower(strings.TrimSpace(scheme)) {
	case "complementary":
		return []float64{0, 180}, nil
	case "analogous":
		return []float64{0, -angle, angle}, nil
	case "triadic":
		return []float64{0, 120, 240}, nil
	case "split-complementary":
		return []float64{0, 180 - angle, 180 + angle}, nil
	case "tetradic":
		return []float64{0, 60, 180, 240}, nil
	case "square":
		return []float64{0, 90, 180, 270}, nil
	default:
		return nil, fmt.Errorf("unknown scheme %q; supported: complementary, analogous, triadic, split-complementary, tetradic, square", scheme)
	}
}

// schemeColors returns a harmony palette derived from base by rotating its OKLCh hue by each of the scheme's offsets, holding lightness and chroma fixed so the colors read as one coherent family. The base color leads the list (canonicalized to hex); alpha is preserved on every entry. Fully deterministic.
func schemeColors(base, scheme string, angle float64) ([]string, error) {
	offsets, err := schemeOffsets(scheme, angle)
	if err != nil {
		return nil, err
	}
	c, alpha, err := parseColor(base)
	if err != nil {
		return nil, err
	}
	l, chroma, hue := oklchOf(c)
	out := make([]string, len(offsets))
	for i, off := range offsets {
		h := math.Mod(hue+off, 360)
		if h < 0 {
			h += 360
		}
		out[i] = hexOut(colorful.OkLch(l, chroma, h), alpha, true, false)
	}
	return out, nil
}

//go:embed descriptions/color_scheme.md
var colorSchemeDescription string

var _ function.Function = (*ColorSchemeFunction)(nil)

type ColorSchemeFunction struct{}

func NewColorSchemeFunction() function.Function { return &ColorSchemeFunction{} }

func (f *ColorSchemeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "color_scheme"
}

func (f *ColorSchemeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Generate a harmony palette (complementary, triadic, ...) from a base color",
		MarkdownDescription: colorSchemeDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "base", Description: "The base color, in any CSS notation. It leads the returned palette."},
			function.StringParameter{Name: "scheme", Description: "One of `complementary`, `analogous`, `triadic`, `split-complementary`, `tetradic`, `square`."},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: "An optional object. Key: `angle` (hue spread in degrees for `analogous` and `split-complementary`, default 30; ignored by the fixed schemes).",
		},
		Return: function.ListReturn{ElementType: types.StringType},
	}
}

func (f *ColorSchemeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var base, scheme string
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &base, &scheme, &optsArgs))
	if resp.Error != nil {
		return
	}
	if unknownListOptionResult(ctx, resp, optsArgs) {
		return
	}
	angle, ferr := schemeAngleOption(optsArgs)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	out, err := schemeColors(base, scheme, angle)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}

// schemeAngleOption parses the optional { angle = N } object, defaulting to 30 degrees.
func schemeAngleOption(opts []types.Dynamic) (float64, *function.FuncError) {
	attrs, ferr := optionsutil.SingleOptionsObject(opts, `{ angle = 45 }`)
	if ferr != nil {
		return defaultSchemeAngle, ferr
	}
	if attrs == nil {
		return defaultSchemeAngle, nil
	}
	for k, v := range attrs {
		if k != "angle" {
			return defaultSchemeAngle, function.NewArgumentFuncError(2, fmt.Sprintf("unknown option key %q; the only key is angle", k))
		}
		num, ok := v.(basetypes.NumberValue)
		if !ok || num.IsNull() {
			return defaultSchemeAngle, function.NewArgumentFuncError(2, "options.angle must be a number")
		}
		f, _ := num.ValueBigFloat().Float64()
		return f, nil
	}
	return defaultSchemeAngle, nil
}
