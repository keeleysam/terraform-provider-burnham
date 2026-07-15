package cedar

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

var _ function.Function = (*CedarFormatFunction)(nil)

type CedarFormatFunction struct{}

func NewCedarFormatFunction() function.Function { return &CedarFormatFunction{} }

func (f *CedarFormatFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cedarformat"
}

func (f *CedarFormatFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Canonicalize a Cedar policy document",
		MarkdownDescription: "Parses a [Cedar](https://www.cedarpolicy.com) policy document and returns its canonical text serialization: normalized layout and indentation, with statements kept in their input order. It fails the plan on syntactically invalid input (use `cedarvalidate` for a non-failing boolean check).\n\nComments are dropped and formatting is normalized (each policy is re-rendered from the parsed AST); annotations such as `@id(...)` are preserved. The output is stable and idempotent, so two documents that differ only in whitespace or comments canonicalize to the same string. Backed by [cedar-go](https://github.com/cedar-policy/cedar-go), the official Go implementation of Cedar.",
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
