package oel

import (
	"context"
	_ "embed"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = (*OELEncodeFunction)(nil)

//go:embed descriptions/oelencode.md
var oelencodeDescription string

type OELEncodeFunction struct{}

func NewOELEncodeFunction() function.Function { return &OELEncodeFunction{} }

func (f *OELEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "oelencode"
}

func (f *OELEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Build an Okta Expression Language string from an HCL data tree",
		MarkdownDescription: oelencodeDescription,
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "expr",
				Description: "The expression tree, in the surface notation described above.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *OELEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var expr types.Dynamic

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &expr))
	if resp.Error != nil {
		return
	}

	if hasUnknown(expr) {
		// A value in the expression is unknown at plan time; return an unknown result so the plan proceeds and the value resolves at apply.
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.StringUnknown()))
		return
	}

	node, err := terraformToNode(expr.UnderlyingValue())
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to read expression: "+err.Error())
		return
	}

	out, err := Encode(node)
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
