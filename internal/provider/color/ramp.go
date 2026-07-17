package color

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	colorful "github.com/lucasb-eyer/go-colorful"
)

// rampColors interpolates n colors evenly across the given stops, in the given blend space. With one stop it returns n copies; with n == 1 it returns the first stop. Alpha is interpolated alongside.
func rampColors(stops []string, n int, space string) ([]string, error) {
	if n < 1 {
		return nil, fmt.Errorf("count must be at least 1, got %d", n)
	}
	if len(stops) == 0 {
		return nil, fmt.Errorf("stops must not be empty")
	}
	fn, err := blendFunc(space)
	if err != nil {
		return nil, err
	}

	colors := make([]colorful.Color, len(stops))
	alphas := make([]float64, len(stops))
	for i, s := range stops {
		c, a, err := parseColor(s)
		if err != nil {
			return nil, fmt.Errorf("stop %d: %w", i, err)
		}
		colors[i], alphas[i] = c, a
	}

	out := make([]string, n)
	m := len(stops)
	for i := 0; i < n; i++ {
		if m == 1 {
			out[i] = hexOut(colors[0], alphas[0], true, false)
			continue
		}
		// Position of output i in [0,1], mapped onto the m-1 segments.
		var p float64
		if n == 1 {
			p = 0
		} else {
			p = float64(i) / float64(n-1)
		}
		pos := p * float64(m-1)
		seg := int(pos)
		local := pos - float64(seg)
		if seg >= m-1 { // clamp the right endpoint
			seg = m - 2
			local = 1
		}
		c := fn(colors[seg], colors[seg+1], local)
		a := alphas[seg]*(1-local) + alphas[seg+1]*local
		out[i] = hexOut(c, a, true, false)
	}
	return out, nil
}

//go:embed descriptions/color_ramp.md
var colorRampDescription string

var _ function.Function = (*ColorRampFunction)(nil)

type ColorRampFunction struct{}

func NewColorRampFunction() function.Function { return &ColorRampFunction{} }

func (f *ColorRampFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "color_ramp"
}

func (f *ColorRampFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Interpolate N colors evenly across a list of stops",
		MarkdownDescription: colorRampDescription,
		Parameters: []function.Parameter{
			function.ListParameter{Name: "stops", ElementType: types.StringType, Description: "Two or more colors to interpolate between (a single stop yields N copies)."},
			function.Int64Parameter{Name: "count", Description: "How many colors to return (>= 1)."},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: "An optional object. Key: `space` (interpolation space, default `oklch`; one of `oklch`, `oklab`, `rgb`, `hsv`, `lab`, `hcl`).",
		},
		Return: function.ListReturn{ElementType: types.StringType},
	}
}

func (f *ColorRampFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var stops types.List
	var count int64
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &stops, &count, &optsArgs))
	if resp.Error != nil {
		return
	}
	if unknownListOptionResult(ctx, resp, optsArgs) {
		return
	}
	var stopStrings []string
	if diags := stops.ElementsAs(ctx, &stopStrings, false); diags.HasError() {
		resp.Error = function.NewArgumentFuncError(0, "stops must be a list of color strings")
		return
	}
	space, ferr := spaceOption(optsArgs, 2)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	out, err := rampColors(stopStrings, int(count), space)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}

// unknownListOptionResult is the list-returning sibling of unknownStringOptionResult.
func unknownListOptionResult(ctx context.Context, resp *function.RunResponse, opts []types.Dynamic) bool {
	for _, o := range opts {
		if hasUnknown(o) {
			resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.ListUnknown(types.StringType)))
			return true
		}
	}
	return false
}
