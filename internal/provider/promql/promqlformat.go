package promql

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/keeleysam/terraform-burnham/internal/provider/optionsutil"
)

//go:embed descriptions/promqlformat.md
var promqlformatDescription string

var _ function.Function = (*PromQLFormatFunction)(nil)

type PromQLFormatFunction struct{}

func NewPromQLFormatFunction() function.Function { return &PromQLFormatFunction{} }

func (f *PromQLFormatFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "promqlformat"
}

func (f *PromQLFormatFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Canonicalize a PromQL expression",
		MarkdownDescription: promqlformatDescription,
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "query",
				Description: "A PromQL expression to canonicalize.",
			},
		},
		VariadicParameter: function.DynamicParameter{
			Name:               "options",
			Description:        "An optional object. Key: `pretty` (bool, default false); when true, return the multi-line indented form instead of a single canonical line.",
			AllowNullValue:     false,
			AllowUnknownValues: false,
		},
		Return: function.StringReturn{},
	}
}

func (f *PromQLFormatFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var query string
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &query, &optsArgs))
	if resp.Error != nil {
		return
	}
	if len(query) > promqlMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("query exceeds maximum supported length of %d bytes", promqlMaxInputBytes))
		return
	}
	if optionsHaveUnknown(optsArgs) {
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.StringUnknown()))
		return
	}

	pretty, ferr := promqlFormatOptions(optsArgs)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	out, err := Format(query, pretty)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, out))
}

// promqlFormatOptions parses the optional { pretty } options object at parameter index 1.
func promqlFormatOptions(opts []types.Dynamic) (bool, *function.FuncError) {
	attrs, ferr := optionsutil.SingleOptionsObject(opts, `{ pretty = true }`)
	if ferr != nil {
		return false, ferr
	}
	pretty := false
	for k, v := range attrs {
		switch k {
		case "pretty":
			b, ok := v.(basetypes.BoolValue)
			if !ok || b.IsNull() || b.IsUnknown() {
				return false, function.NewArgumentFuncError(1, "options.pretty must be a boolean")
			}
			pretty = b.ValueBool()
		default:
			return false, function.NewArgumentFuncError(1, fmt.Sprintf("unknown option key %q; the only supported key is pretty", k))
		}
	}
	return pretty, nil
}

// optionsHaveUnknown reports whether the variadic options object holds any unknown value.
func optionsHaveUnknown(opts []types.Dynamic) bool {
	for _, o := range opts {
		if hasUnknown(o) {
			return true
		}
	}
	return false
}
