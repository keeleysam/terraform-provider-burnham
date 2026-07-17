package color

import (
	"context"
	_ "embed"
	"fmt"
	"math"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/keeleysam/terraform-burnham/internal/provider/optionsutil"
	colorful "github.com/lucasb-eyer/go-colorful"
)

// distinctDefaults are the OKLCh anchors for color_distinct. Lightness and chroma stay fixed while hue sweeps the wheel, so the colors read as a coherent family at similar perceived weight. Chosen to be vivid but in-gamut across the whole hue circle.
const (
	distinctLightness = 0.72
	distinctChroma    = 0.14
)

type distinctOpts struct {
	lightness float64
	chroma    float64
	hueOffset float64
}

func defaultDistinctOpts() distinctOpts {
	return distinctOpts{lightness: distinctLightness, chroma: distinctChroma, hueOffset: 0}
}

// distinctColors returns n colors whose OKLCh hues are spread evenly around the wheel at a fixed lightness and chroma, starting from hueOffset. It is fully deterministic (no randomness), so the same n always yields the same colors and plan output never churns.
func distinctColors(n int, opts distinctOpts) ([]string, error) {
	if n < 1 {
		return nil, fmt.Errorf("count must be at least 1, got %d", n)
	}
	out := make([]string, n)
	for i := 0; i < n; i++ {
		hue := math.Mod(opts.hueOffset+float64(i)*360.0/float64(n), 360)
		if hue < 0 {
			hue += 360
		}
		c := colorful.OkLch(opts.lightness, opts.chroma, hue)
		out[i] = hexOut(c, 1, true, false)
	}
	return out, nil
}

//go:embed descriptions/color_distinct.md
var colorDistinctDescription string

var _ function.Function = (*ColorDistinctFunction)(nil)

type ColorDistinctFunction struct{}

func NewColorDistinctFunction() function.Function { return &ColorDistinctFunction{} }

func (f *ColorDistinctFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "color_distinct"
}

func (f *ColorDistinctFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Generate N deterministic, maximally-distinct colors",
		MarkdownDescription: colorDistinctDescription,
		Parameters: []function.Parameter{
			function.Int64Parameter{Name: "count", Description: "How many distinct colors to generate (>= 1)."},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: "An optional object. Keys: `lightness` (OKLCh L, 0-1, default 0.72), `chroma` (OKLCh C, default 0.14), and `hue_offset` (degrees to rotate the starting hue, default 0).",
		},
		Return: function.ListReturn{ElementType: types.StringType},
	}
}

func (f *ColorDistinctFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var count int64
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &count, &optsArgs))
	if resp.Error != nil {
		return
	}
	if unknownListOptionResult(ctx, resp, optsArgs) {
		return
	}
	opts, ferr := parseDistinctOpts(optsArgs)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	out, err := distinctColors(int(count), opts)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}

func parseDistinctOpts(opts []types.Dynamic) (distinctOpts, *function.FuncError) {
	out := defaultDistinctOpts()
	attrs, ferr := optionsutil.SingleOptionsObject(opts, `{ lightness = 0.7, chroma = 0.15, hue_offset = 30 }`)
	if ferr != nil {
		return out, ferr
	}
	for k, v := range attrs {
		val, err := numberAttr(v)
		if err != nil {
			return out, function.NewArgumentFuncError(1, fmt.Sprintf("options.%s must be a number", k))
		}
		switch k {
		case "lightness":
			out.lightness = val
		case "chroma":
			out.chroma = val
		case "hue_offset":
			out.hueOffset = val
		default:
			return out, function.NewArgumentFuncError(1, fmt.Sprintf("unknown option key %q; supported keys are lightness, chroma, hue_offset", k))
		}
	}
	return out, nil
}

// numberAttr extracts a float64 from a Terraform Number attribute.
func numberAttr(v attr.Value) (float64, error) {
	num, ok := v.(basetypes.NumberValue)
	if !ok || num.IsNull() {
		return 0, fmt.Errorf("not a number")
	}
	f, _ := num.ValueBigFloat().Float64()
	return f, nil
}
