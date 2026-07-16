package cel

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = (*CELFormatFunction)(nil)

//go:embed descriptions/celformat.md
var celformatDescription string

type CELFormatFunction struct{}

func NewCELFormatFunction() function.Function { return &CELFormatFunction{} }

func (f *CELFormatFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "celformat"
}

func (f *CELFormatFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Canonicalize and pretty-print a hand-written CEL expression",
		MarkdownDescription: celformatDescription,
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "expr",
				Description: "A CEL expression string.",
			},
		},
		VariadicParameter: function.DynamicParameter{
			Name:               "options",
			Description:        "An optional object. The only key is `format`, an object mirroring cel-go's unparser options plus celfmt's pretty-printing (each optional; omitted keys use the backend default): `pretty` (bool) indents structural containers (lists, maps, call/macro arguments, parenthesized groups) onto multiple lines; `indent` (string, default a tab) is the indent unit; `always_comma` (bool) adds trailing commas; `wrap_on_column` (number) sets the width used to introduce line breaks (default 80 when pretty); `wrap_on_operators` (list, default `[\"&&\", \"||\"]`) and `wrap_after_column_limit` (bool, default true) control operator wrapping. Example: `{ format = { pretty = true, indent = \"  \" } }`. With no `format`, the output is a single canonical line. Note: boolean operator chains are width-wrapped, not one-per-line.",
			AllowNullValue:     false,
			AllowUnknownValues: false,
		},
		Return: function.StringReturn{},
	}
}

func (f *CELFormatFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var expr string
	var optsArgs []types.Dynamic

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &expr, &optsArgs))
	if resp.Error != nil {
		return
	}

	if len(expr) > celMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("expression exceeds maximum supported length of %d bytes", celMaxInputBytes))
		return
	}
	if optionsHaveUnknown(optsArgs) {
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.StringUnknown()))
		return
	}

	formatOpts, ferr := celFormatOptions(optsArgs)
	if ferr != nil {
		resp.Error = ferr
		return
	}

	out, err := Format(expr, formatOpts...)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, out))
}
