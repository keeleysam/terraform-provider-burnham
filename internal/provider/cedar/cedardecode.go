package cedar

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = (*CedarDecodeFunction)(nil)

type CedarDecodeFunction struct{}

func NewCedarDecodeFunction() function.Function { return &CedarDecodeFunction{} }

func (f *CedarDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cedardecode"
}

func (f *CedarDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Decode a Cedar policy into its JSON (EST) data tree",
		MarkdownDescription: "Parses a single [Cedar](https://www.cedarpolicy.com) policy in its human-readable text form and returns Cedar's JSON policy format, the EST, as a data tree. It is the inverse of `cedarencode`, so `cedarencode(cedardecode(x))` round-trips to the canonical form of `x`. Useful for inspecting, querying, or patching a hand-written policy as structured data, and for discovering the EST shape to feed back into `cedarencode`.\n\nIt handles exactly one policy statement (the shape of an `aws_verifiedpermissions_policy` static policy); a document with several policies is a policy set and is rejected here (use `cedarformat` or `cedarvalidate` for those). The return is a dynamic value ready to `jsonencode` into Cedar's JSON policy format. Backed by [cedar-go](https://github.com/cedar-policy/cedar-go), the official Go implementation of Cedar.",
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
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("policy document exceeds maximum supported length of %d bytes", cedarMaxInputBytes))
		return
	}

	tree, err := Decode(policy)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}
	value, err := nodeToAttr(tree)
	if err != nil {
		// Post-parse internal conversion failure, not an invalid argument.
		resp.Error = function.NewFuncError(err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(value)))
}
