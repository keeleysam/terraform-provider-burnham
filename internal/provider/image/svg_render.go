package image

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/keeleysam/terraform-burnham/internal/provider/image/resvg"
	"github.com/keeleysam/terraform-burnham/internal/provider/optionsutil"
)

// bundledFonts are always available to the renderer. The real design will bundle
// DejaVu Sans/Serif/Mono + Liberation via the go-fonts modules; for now we ship
// the Go font (text) plus Noto Color Emoji COLRv1 (vector color emoji), both
// permissive. resvg renders the COLRv1 build natively.
var bundledFonts = [][]byte{goregular.TTF, notoColorEmoji}

var _ function.Function = (*SVGRenderFunction)(nil)

type SVGRenderFunction struct{}

func NewSVGRenderFunction() function.Function { return &SVGRenderFunction{} }

func (f *SVGRenderFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "svg_render"
}

func (f *SVGRenderFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Render an SVG document to a base64-encoded PNG",
		MarkdownDescription: "Rasterizes an SVG document to a PNG, returned as a base64 string. Renders gradients, clipping, masks, filters, text, and native color emoji at near-browser fidelity via resvg (run as WebAssembly), fully deterministically and with no system-font access. Options: `width` / `height` (output pixels; supply one and the other follows the SVG aspect ratio, supply neither to use the SVG's intrinsic size), `scale` (multiplier over the intrinsic size), and `fonts` (a list of additional base64-encoded TTF/OTF fonts to load, for scripts or brand fonts beyond the bundled set).",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "svg", Description: "The SVG document to render."},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: "An optional object with keys: `width`, `height`, `scale` (numbers), and `fonts` (list of base64-encoded fonts).",
		},
		Return: function.StringReturn{},
	}
}

func (f *SVGRenderFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var svg string
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &svg, &optsArgs))
	if resp.Error != nil {
		return
	}

	width, height, scale, fonts, ferr := parseRenderOptions(optsArgs)
	if ferr != nil {
		resp.Error = ferr
		return
	}

	all := make([][]byte, 0, len(bundledFonts)+len(fonts))
	all = append(all, bundledFonts...)
	all = append(all, fonts...)

	png, err := resvg.Render(ctx, []byte(svg), all, width, height, scale)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	out := base64.StdEncoding.EncodeToString(png)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}

func parseRenderOptions(opts []types.Dynamic) (width, height uint32, scale float32, fonts [][]byte, ferr *function.FuncError) {
	attrs, ferr := optionsutil.SingleOptionsObject(opts, `{ width = 512 }`)
	if ferr != nil {
		return 0, 0, 0, nil, ferr
	}
	for k, v := range attrs {
		switch k {
		case "width":
			n, err := optionsutil.NumberAttrToInt(v)
			if err != nil || n < 0 {
				return 0, 0, 0, nil, function.NewArgumentFuncError(1, "options.width must be a non-negative whole number")
			}
			width = uint32(n)
		case "height":
			n, err := optionsutil.NumberAttrToInt(v)
			if err != nil || n < 0 {
				return 0, 0, 0, nil, function.NewArgumentFuncError(1, "options.height must be a non-negative whole number")
			}
			height = uint32(n)
		case "scale":
			num, ok := v.(basetypes.NumberValue)
			if !ok || num.IsNull() {
				return 0, 0, 0, nil, function.NewArgumentFuncError(1, "options.scale must be a number")
			}
			sf, _ := num.ValueBigFloat().Float64()
			scale = float32(sf)
		case "fonts":
			list, err := base64StringList(v)
			if err != nil {
				return 0, 0, 0, nil, function.NewArgumentFuncError(1, "options.fonts must be a list of base64-encoded fonts: "+err.Error())
			}
			fonts = list
		default:
			return 0, 0, 0, nil, function.NewArgumentFuncError(1, fmt.Sprintf("unknown option key %q; supported keys are width, height, scale, fonts", k))
		}
	}
	return width, height, scale, fonts, nil
}

// base64StringList decodes a Terraform list/tuple of base64 strings into raw bytes.
func base64StringList(v attr.Value) ([][]byte, error) {
	var elems []attr.Value
	switch val := v.(type) {
	case basetypes.ListValue:
		elems = val.Elements()
	case basetypes.TupleValue:
		elems = val.Elements()
	case basetypes.SetValue:
		elems = val.Elements()
	default:
		return nil, fmt.Errorf("not a list")
	}
	out := make([][]byte, 0, len(elems))
	for i, e := range elems {
		s, ok := e.(basetypes.StringValue)
		if !ok || s.IsNull() {
			return nil, fmt.Errorf("element %d is not a string", i)
		}
		b, err := base64.StdEncoding.DecodeString(s.ValueString())
		if err != nil {
			return nil, fmt.Errorf("element %d is not valid base64", i)
		}
		out = append(out, b)
	}
	return out, nil
}
