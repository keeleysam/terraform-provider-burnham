package cedar

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = (*CedarEncodeFunction)(nil)

type CedarEncodeFunction struct{}

func NewCedarEncodeFunction() function.Function { return &CedarEncodeFunction{} }

func (f *CedarEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cedarencode"
}

func (f *CedarEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Build a Cedar policy from its JSON (EST) data tree",
		MarkdownDescription: "Builds a single [Cedar](https://www.cedarpolicy.com) policy in its human-readable text form from a structured HCL value, so you assemble a policy from Terraform data with no string templating. The result is a canonical policy statement suitable for `aws_verifiedpermissions_policy`.\n\nThe input is Cedar's own JSON policy format, the EST: an object with `effect` (`\"permit\"` or `\"forbid\"`), `principal`, `action`, `resource`, and an optional `conditions` list. Cedar defines this JSON format directly, so the HCL mirrors it one-to-one, and `cedardecode` produces the same shape. Each of `principal`, `action`, and `resource` is a scope object `{ op = ..., entity = { type = ..., id = ... } }` where `op` is `\"==\"`, `\"in\"`, `\"is\"`, or `\"All\"` (an unconstrained scope, e.g. a bare `resource`). Each `conditions` entry is a `when`/`unless` clause in Cedar's EST expression form, a nested AST; the simplest way to get that shape for a non-trivial policy is to write it as text and run it through `cedardecode`.\n\nThe tree is validated as it is converted, so `cedarencode` never emits a syntactically invalid policy, and the output is canonical (byte-identical to what `cedarformat` produces for one policy). Backed by [cedar-go](https://github.com/cedar-policy/cedar-go), the official Go implementation of Cedar.",
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
