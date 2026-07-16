package cedar

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

//go:embed descriptions/cedarformat.md
var cedarformatDescription string

var _ function.Function = (*CedarFormatFunction)(nil)

type CedarFormatFunction struct{}

func NewCedarFormatFunction() function.Function { return &CedarFormatFunction{} }

func (f *CedarFormatFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cedarformat"
}

func (f *CedarFormatFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Canonicalize a Cedar policy document",
		MarkdownDescription: cedarformatDescription,
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "policies",
				Description: "A Cedar policy document to canonicalize.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *CedarFormatFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var policies string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &policies))
	if resp.Error != nil {
		return
	}
	if len(policies) > cedarMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("policy document exceeds maximum supported length of %d bytes", cedarMaxInputBytes))
		return
	}
	out, err := Format(policies)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, out))
}
