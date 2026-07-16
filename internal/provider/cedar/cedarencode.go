package cedar

import (
	"context"
	_ "embed"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

//go:embed descriptions/cedarencode.md
var cedarencodeDescription string

var _ function.Function = (*CedarEncodeFunction)(nil)

type CedarEncodeFunction struct{}

func NewCedarEncodeFunction() function.Function { return &CedarEncodeFunction{} }

func (f *CedarEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cedarencode"
}

func (f *CedarEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Build a Cedar policy from its JSON (EST) data tree",
		MarkdownDescription: cedarencodeDescription,
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "policy",
				Description: "The policy as a Cedar EST (JSON policy format) data tree.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *CedarEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var policy types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &policy))
	if resp.Error != nil {
		return
	}
	if hasUnknown(policy) {
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.StringUnknown()))
		return
	}

	tree, err := terraformToNode(policy.UnderlyingValue())
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to read policy: "+err.Error())
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
