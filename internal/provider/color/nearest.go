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

// distanceMetrics are the perceptual-distance functions color_nearest accepts. CIEDE2000 is the default: it is the most accurate model of perceived color difference. The others are simpler escape hatches for callers who want a specific space.
var distanceMetrics = map[string]func(a, b colorful.Color) float64{
	"ciede2000": func(a, b colorful.Color) float64 { return a.DistanceCIEDE2000(b) },
	"lab":       func(a, b colorful.Color) float64 { return a.DistanceLab(b) },
	"rgb":       func(a, b colorful.Color) float64 { return a.DistanceRgb(b) },
	"oklab": func(a, b colorful.Color) float64 {
		al, aa, ab := a.OkLab()
		bl, ba, bb := b.OkLab()
		return math.Sqrt((al-bl)*(al-bl) + (aa-ba)*(aa-ba) + (ab-bb)*(ab-bb))
	},
}

func distanceFunc(metric string) (func(a, b colorful.Color) float64, error) {
	fn, ok := distanceMetrics[strings.ToLower(strings.TrimSpace(metric))]
	if !ok {
		return nil, fmt.Errorf("unknown metric %q; supported: ciede2000, oklab, lab, rgb", metric)
	}
	return fn, nil
}

// nearestColor returns the palette entry perceptually closest to color under the given metric, returning it exactly as supplied (not canonicalized) so a match keeps its original spelling. On a tie the earlier entry wins.
func nearestColor(color string, palette []string, metric string) (string, error) {
	if len(palette) == 0 {
		return "", fmt.Errorf("palette must not be empty")
	}
	dist, err := distanceFunc(metric)
	if err != nil {
		return "", err
	}
	target, _, err := parseColor(color)
	if err != nil {
		return "", err
	}
	best, bestDist := "", math.Inf(1)
	for _, entry := range palette {
		c, _, err := parseColor(entry)
		if err != nil {
			return "", fmt.Errorf("palette entry %q: could not parse", entry)
		}
		if d := dist(target, c); d < bestDist {
			best, bestDist = entry, d
		}
	}
	return best, nil
}

//go:embed descriptions/color_nearest.md
var colorNearestDescription string

var _ function.Function = (*ColorNearestFunction)(nil)

type ColorNearestFunction struct{}

func NewColorNearestFunction() function.Function { return &ColorNearestFunction{} }

func (f *ColorNearestFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "color_nearest"
}

func (f *ColorNearestFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Snap a color to the perceptually closest entry in a palette",
		MarkdownDescription: colorNearestDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "color", Description: "The color to match, in any CSS notation."},
			function.ListParameter{Name: "palette", ElementType: types.StringType, Description: "The colors to choose from. The closest one is returned exactly as written."},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: "An optional object. Key: `metric` (distance model, default `ciede2000`; one of `ciede2000`, `oklab`, `lab`, `rgb`).",
		},
		Return: function.StringReturn{},
	}
}

func (f *ColorNearestFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var color string
	var palette types.List
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &color, &palette, &optsArgs))
	if resp.Error != nil {
		return
	}
	if unknownStringOptionResult(ctx, resp, optsArgs) {
		return
	}
	var paletteStrings []string
	if diags := palette.ElementsAs(ctx, &paletteStrings, false); diags.HasError() {
		resp.Error = function.NewArgumentFuncError(1, "palette must be a list of color strings")
		return
	}
	metric, ferr := metricOption(optsArgs)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	out, err := nearestColor(color, paletteStrings, metric)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}

// metricOption parses the optional { metric = "..." } object, defaulting to ciede2000.
func metricOption(opts []types.Dynamic) (string, *function.FuncError) {
	attrs, ferr := optionsutil.SingleOptionsObject(opts, `{ metric = "oklab" }`)
	if ferr != nil {
		return "ciede2000", ferr
	}
	if attrs == nil {
		return "ciede2000", nil
	}
	for k, v := range attrs {
		if k != "metric" {
			return "ciede2000", function.NewArgumentFuncError(2, fmt.Sprintf("unknown option key %q; the only key is metric", k))
		}
		s, ok := v.(basetypes.StringValue)
		if !ok || s.IsNull() {
			return "ciede2000", function.NewArgumentFuncError(2, "options.metric must be a string")
		}
		return s.ValueString(), nil
	}
	return "ciede2000", nil
}
