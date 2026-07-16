package oel

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

var _ function.Function = (*OELValidateFunction)(nil)

//go:embed descriptions/oelvalidate.md
var oelvalidateDescription string

type OELValidateFunction struct{}

func NewOELValidateFunction() function.Function { return &OELValidateFunction{} }

func (f *OELValidateFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "oelvalidate"
}

func (f *OELValidateFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Report whether a string is a syntactically valid Okta EL expression",
		MarkdownDescription: oelvalidateDescription,
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "expr",
				Description: "An Okta EL expression string to check.",
			},
		},
		Return: function.BoolReturn{},
	}
}

func (f *OELValidateFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var expr string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &expr))
	if resp.Error != nil {
		return
	}
	if len(expr) > oelMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("expression exceeds maximum supported length of %d bytes", oelMaxInputBytes))
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, IsValid(expr)))
}
