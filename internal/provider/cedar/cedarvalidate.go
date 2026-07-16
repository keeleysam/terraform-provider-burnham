package cedar

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

//go:embed descriptions/cedarvalidate.md
var cedarvalidateDescription string

var _ function.Function = (*CedarValidateFunction)(nil)

type CedarValidateFunction struct{}

func NewCedarValidateFunction() function.Function { return &CedarValidateFunction{} }

func (f *CedarValidateFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cedarvalidate"
}

func (f *CedarValidateFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Report whether a string is a syntactically valid Cedar policy document",
		MarkdownDescription: cedarvalidateDescription,
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "policies",
				Description: "A Cedar policy document to check.",
			},
		},
		Return: function.BoolReturn{},
	}
}

func (f *CedarValidateFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var policies string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &policies))
	if resp.Error != nil {
		return
	}
	if len(policies) > cedarMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("policy document exceeds maximum supported length of %d bytes", cedarMaxInputBytes))
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, IsValid(policies)))
}
