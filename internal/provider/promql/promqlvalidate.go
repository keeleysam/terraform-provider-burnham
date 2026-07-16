package promql

import (
	"context"
	_ "embed"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

//go:embed descriptions/promqlvalidate.md
var promqlvalidateDescription string

var _ function.Function = (*PromQLValidateFunction)(nil)

type PromQLValidateFunction struct{}

func NewPromQLValidateFunction() function.Function { return &PromQLValidateFunction{} }

func (f *PromQLValidateFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "promqlvalidate"
}

func (f *PromQLValidateFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Report whether a string is a valid PromQL expression",
		MarkdownDescription: promqlvalidateDescription,
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "query",
				Description: "A PromQL expression to check.",
			},
		},
		Return: function.BoolReturn{},
	}
}

func (f *PromQLValidateFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var query string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &query))
	if resp.Error != nil {
		return
	}
	if len(query) > promqlMaxInputBytes {
		// Over the size guard, report not-valid rather than failing the plan, keeping the "never fails" contract absolute (a query this large is not a real one).
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, false))
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, IsValid(query)))
}
