package promql

import (
	"context"
	_ "embed"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

//go:embed descriptions/promqlencode.md
var promqlencodeDescription string

var _ function.Function = (*PromQLEncodeFunction)(nil)

type PromQLEncodeFunction struct{}

func NewPromQLEncodeFunction() function.Function { return &PromQLEncodeFunction{} }

func (f *PromQLEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "promqlencode"
}

func (f *PromQLEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Build a PromQL query from an HCL data tree",
		MarkdownDescription: promqlencodeDescription,
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "query",
				Description: "The query as a data tree mirroring the PromQL AST.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *PromQLEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var query types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &query))
	if resp.Error != nil {
		return
	}
	if hasUnknown(query) {
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.StringUnknown()))
		return
	}

	tree, err := terraformToNode(query.UnderlyingValue())
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to read query: "+err.Error())
		return
	}
	out, err := Encode(tree)
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
