package promql

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

//go:embed descriptions/promqldecode.md
var promqldecodeDescription string

var _ function.Function = (*PromQLDecodeFunction)(nil)

type PromQLDecodeFunction struct{}

func NewPromQLDecodeFunction() function.Function { return &PromQLDecodeFunction{} }

func (f *PromQLDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "promqldecode"
}

func (f *PromQLDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Parse a PromQL query into a promqlencode data tree",
		MarkdownDescription: promqldecodeDescription,
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "query",
				Description: "A PromQL expression to parse into a data tree.",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *PromQLDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var query string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &query))
	if resp.Error != nil {
		return
	}
	if len(query) > promqlMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("query exceeds maximum supported length of %d bytes", promqlMaxInputBytes))
		return
	}

	tree, err := Decode(query)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}
	value, err := nodeToAttr(tree)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(value)))
}
