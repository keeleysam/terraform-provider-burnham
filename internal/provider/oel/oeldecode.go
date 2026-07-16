package oel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = (*OELDecodeFunction)(nil)

type OELDecodeFunction struct{}

func NewOELDecodeFunction() function.Function { return &OELDecodeFunction{} }

func (f *OELDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "oeldecode"
}

func (f *OELDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Decode an Okta EL expression string into an oelencode data tree",
		MarkdownDescription: "Parses an [Okta Expression Language](https://developer.okta.com/docs/reference/okta-expression-language/) string and returns it as the HCL data tree that `oelencode` consumes, so `provider::burnham::oelencode(provider::burnham::oeldecode(expr))` round-trips to the canonical form of `expr`. Primarily a tool for testing and for migrating hand-written expressions into the data model.\n\nReferences decode to `{ ident = \"...\" }`, operators to their token keys, and calls to the `call` forms; a dotted path that embeds a group-membership method hop, which has no direct surface form, decodes to a `{ raw = \"...\" }` escape that `oelencode` re-parses. The return is a dynamic value; list literals decode to Terraform tuples (heterogeneous), which `oelencode` accepts on the way back. Covers the full documented grammar. Backed by [okta-expression-parser](https://github.com/keeleysam/okta-expression-parser).",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "expr",
				Description: "An Okta EL expression string.",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *OELDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var expr string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &expr))
	if resp.Error != nil {
		return
	}
	if len(expr) > oelMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("expression exceeds maximum supported length of %d bytes", oelMaxInputBytes))
		return
	}

	node, err := Decode(expr)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}
	value, err := nodeToAttr(node)
	if err != nil {
		// Post-parse internal conversion failure, not an invalid argument.
		resp.Error = function.NewFuncError(err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(value)))
}
