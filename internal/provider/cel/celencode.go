package cel

import (
	"context"
	_ "embed"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

//go:embed descriptions/celencode.md
var celencodeDescription string

var _ function.Function = (*CELEncodeFunction)(nil)

/*
TODO (deferred): support emitting CEL comments from the data tree, e.g. a per-node `comment` field that renders as `// ...` in the output.

This does not work today and cannot be bolted on with the current backends. cel-go's AST has no comment representation (its `common/ast` exposes none, the parser has no comment-retention option, and the unparser never emits comments), so a comment attached to a node has nowhere to live once we build the cel-go AST. celfmt does not read comments from the AST either; it recovers them only by scraping the original source text by line position, which a built (source-less) expression has nothing for. Our pretty pipeline also reparses a canonical, comment-free seed before celfmt runs, stripping any comment regardless.

Supporting per-node/structural comments therefore requires our own emitter (the deferred custom formatter): walk the tree and write `// ...` inline while laying out the expression, delegating leaf rendering to cel-go. A single whole-expression leading/trailing comment, by contrast, is trivial string concatenation and needs no AST/library support.
*/
type CELEncodeFunction struct{}

func NewCELEncodeFunction() function.Function { return &CELEncodeFunction{} }

func (f *CELEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "celencode"
}

func (f *CELEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Build a CEL expression string from an HCL data tree",
		MarkdownDescription: celencodeDescription,
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "expr",
				Description: "The expression tree, in the surface or canonical notation (mixable).",
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

func (f *CELEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var expr types.Dynamic
	var optsArgs []types.Dynamic

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &expr, &optsArgs))
	if resp.Error != nil {
		return
	}

	if hasUnknown(expr) || optionsHaveUnknown(optsArgs) {
		// A value in the expression or options is unknown at plan time; return an unknown result so the plan proceeds and the value resolves at apply.
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.StringUnknown()))
		return
	}

	formatOpts, ferr := celFormatOptions(optsArgs)
	if ferr != nil {
		resp.Error = ferr
		return
	}

	node, err := terraformToNode(expr.UnderlyingValue())
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to read expression: "+err.Error())
		return
	}

	out, err := Encode(node, formatOpts...)
	if errors.Is(err, errInvalidOutput) {
		resp.Error = function.NewFuncError(err.Error())
		return
	}
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, out))
}
