package cedar

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

//go:embed descriptions/cedardecode.md
var cedardecodeDescription string

var _ function.Function = (*CedarDecodeFunction)(nil)

type CedarDecodeFunction struct{}

func NewCedarDecodeFunction() function.Function { return &CedarDecodeFunction{} }

func (f *CedarDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cedardecode"
}

func (f *CedarDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Decode a Cedar policy into its JSON (EST) data tree",
		MarkdownDescription: cedardecodeDescription,
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "policy",
				Description: "A single Cedar policy in the DSL syntax.",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *CedarDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var policy string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &policy))
	if resp.Error != nil {
		return
	}
	if len(policy) > cedarMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("policy exceeds maximum supported length of %d bytes", cedarMaxInputBytes))
		return
	}

	tree, err := Decode(policy)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}
	value, err := nodeToAttr(tree)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(value)))
}
