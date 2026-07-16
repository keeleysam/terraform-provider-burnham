package oel

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

var _ function.Function = (*OELFormatFunction)(nil)

//go:embed descriptions/oelformat.md
var oelformatDescription string

type OELFormatFunction struct{}

func NewOELFormatFunction() function.Function { return &OELFormatFunction{} }

func (f *OELFormatFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "oelformat"
}

func (f *OELFormatFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Canonicalize an Okta EL expression string",
		MarkdownDescription: oelformatDescription,
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "expr",
				Description: "An Okta EL expression string to canonicalize.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *OELFormatFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var expr string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &expr))
	if resp.Error != nil {
		return
	}
	if len(expr) > oelMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("expression exceeds maximum supported length of %d bytes", oelMaxInputBytes))
		return
	}
	out, err := Format(expr)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, out))
}
