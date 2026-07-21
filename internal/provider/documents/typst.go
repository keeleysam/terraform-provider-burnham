package documents

import (
	"context"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/keeleysam/terraform-burnham/internal/provider/documents/typst"
	"github.com/keeleysam/terraform-burnham/internal/provider/image"
	"github.com/keeleysam/terraform-burnham/internal/provider/optionsutil"
)

// typstOptions holds the parsed contents of the shared options object.
type typstOptions struct {
	inputs any
	files  map[string]string
	fonts  [][]byte
	ppi    float64
}

// parseTypstOptions reads the (zero-or-one) options object. allowPPI is set only for typst_png, the
// one format with a raster resolution.
func parseTypstOptions(opts []types.Dynamic, allowPPI bool) (typstOptions, *function.FuncError) {
	var o typstOptions
	attrs, ferr := optionsutil.SingleOptionsObject(opts, `{ inputs = { name = "Ada" } }`)
	if ferr != nil {
		return o, ferr
	}
	for k, v := range attrs {
		switch k {
		case "inputs":
			g, err := optionsutil.AttrToGo(v)
			if err != nil {
				return o, function.NewArgumentFuncError(1, "options.inputs: "+err.Error())
			}
			o.inputs = g
		case "files":
			m, err := optionsutil.Base64StringMap(v)
			if err != nil {
				return o, function.NewArgumentFuncError(1, "options.files must be a map of base64-encoded file contents: "+err.Error())
			}
			o.files = m
		case "fonts":
			list, err := optionsutil.Base64List(v)
			if err != nil {
				return o, function.NewArgumentFuncError(1, "options.fonts must be a list of base64-encoded fonts: "+err.Error())
			}
			o.fonts = list
		case "ppi":
			if !allowPPI {
				return o, function.NewArgumentFuncError(1, "options.ppi is only supported by typst_png")
			}
			num, ok := v.(basetypes.NumberValue)
			if !ok || num.IsNull() {
				return o, function.NewArgumentFuncError(1, "options.ppi must be a number")
			}
			f, _ := num.ValueBigFloat().Float64()
			o.ppi = f
		default:
			return o, function.NewArgumentFuncError(1, fmt.Sprintf("unknown option key %q; supported keys are inputs, files, fonts%s", k, ppiHint(allowPPI)))
		}
	}
	return o, nil
}

func ppiHint(allowPPI bool) string {
	if allowPPI {
		return ", ppi"
	}
	return ""
}

// renderTypst runs the engine with the bundled fonts (plus any user fonts) and maps engine errors to
// the right diagnostic: a document that fails to compile points at the source (argument 0); anything
// else is an internal fault reported generally.
func renderTypst(ctx context.Context, op, source string, o typstOptions) ([][]byte, *function.FuncError) {
	fonts := append(image.BundledFonts(), o.fonts...)
	pages, err := typst.Render(ctx, typst.Request{
		Op:     op,
		Source: source,
		Inputs: o.inputs,
		Files:  o.files,
		Fonts:  fonts,
		PPI:    o.ppi,
	})
	if err != nil {
		var ee *typst.EngineError
		if errors.As(err, &ee) {
			return nil, function.NewArgumentFuncError(0, ee.Msg)
		}
		return nil, function.NewFuncError(err.Error())
	}
	return pages, nil
}

// ── typst_pdf ───────────────────────────────────────────────────

//go:embed descriptions/typst_pdf.md
var typstPDFDescription string

var _ function.Function = (*TypstPDFFunction)(nil)

type TypstPDFFunction struct{}

func NewTypstPDFFunction() function.Function { return &TypstPDFFunction{} }

func (f *TypstPDFFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "typst_pdf"
}

func (f *TypstPDFFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Typeset a Typst document to a base64-encoded PDF",
		MarkdownDescription: typstPDFDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "source", Description: "The Typst markup to typeset."},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: "An optional object with keys: `inputs` (structured data exposed as `sys.inputs`), `files` (map of path to base64-encoded imports/assets), and `fonts` (list of base64-encoded fonts).",
		},
		Return: function.StringReturn{},
	}
}

func (f *TypstPDFFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var source string
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &source, &optsArgs))
	if resp.Error != nil {
		return
	}
	o, ferr := parseTypstOptions(optsArgs, false)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	pages, ferr := renderTypst(ctx, "pdf", source, o)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	out := base64.StdEncoding.EncodeToString(pages[0])
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, out))
}

// ── typst_png ───────────────────────────────────────────────────

//go:embed descriptions/typst_png.md
var typstPNGDescription string

var _ function.Function = (*TypstPNGFunction)(nil)

type TypstPNGFunction struct{}

func NewTypstPNGFunction() function.Function { return &TypstPNGFunction{} }

func (f *TypstPNGFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "typst_png"
}

func (f *TypstPNGFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Render a Typst document to base64-encoded PNGs, one per page",
		MarkdownDescription: typstPNGDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "source", Description: "The Typst markup to render."},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: "An optional object with keys: `inputs`, `files`, `fonts`, and `ppi` (output resolution in pixels per inch; default 144).",
		},
		Return: function.ListReturn{ElementType: types.StringType},
	}
}

func (f *TypstPNGFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var source string
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &source, &optsArgs))
	if resp.Error != nil {
		return
	}
	o, ferr := parseTypstOptions(optsArgs, true)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	pages, ferr := renderTypst(ctx, "png", source, o)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	out := make([]string, len(pages))
	for i, p := range pages {
		out[i] = base64.StdEncoding.EncodeToString(p)
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, out))
}

// ── typst_svg ───────────────────────────────────────────────────

//go:embed descriptions/typst_svg.md
var typstSVGDescription string

var _ function.Function = (*TypstSVGFunction)(nil)

type TypstSVGFunction struct{}

func NewTypstSVGFunction() function.Function { return &TypstSVGFunction{} }

func (f *TypstSVGFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "typst_svg"
}

func (f *TypstSVGFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Render a Typst document to SVG strings, one per page",
		MarkdownDescription: typstSVGDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "source", Description: "The Typst markup to render."},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: "An optional object with keys: `inputs`, `files`, and `fonts`.",
		},
		Return: function.ListReturn{ElementType: types.StringType},
	}
}

func (f *TypstSVGFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var source string
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &source, &optsArgs))
	if resp.Error != nil {
		return
	}
	o, ferr := parseTypstOptions(optsArgs, false)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	pages, ferr := renderTypst(ctx, "svg", source, o)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	// SVG is text, so return it directly rather than base64.
	out := make([]string, len(pages))
	for i, p := range pages {
		out[i] = string(p)
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, out))
}

// ── typst_html ──────────────────────────────────────────────────

//go:embed descriptions/typst_html.md
var typstHTMLDescription string

var _ function.Function = (*TypstHTMLFunction)(nil)

type TypstHTMLFunction struct{}

func NewTypstHTMLFunction() function.Function { return &TypstHTMLFunction{} }

func (f *TypstHTMLFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "typst_html"
}

func (f *TypstHTMLFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Export a Typst document to a self-contained HTML string (experimental)",
		MarkdownDescription: typstHTMLDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "source", Description: "The Typst markup to export."},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: "An optional object with keys: `inputs`, `files`, and `fonts`.",
		},
		Return: function.StringReturn{},
	}
}

func (f *TypstHTMLFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var source string
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &source, &optsArgs))
	if resp.Error != nil {
		return
	}
	o, ferr := parseTypstOptions(optsArgs, false)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	pages, ferr := renderTypst(ctx, "html", source, o)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	// HTML is text, returned directly.
	out := string(pages[0])
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, out))
}
