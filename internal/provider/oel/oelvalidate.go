package oel

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

var _ function.Function = (*OELValidateFunction)(nil)

type OELValidateFunction struct{}

func NewOELValidateFunction() function.Function { return &OELValidateFunction{} }

func (f *OELValidateFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "oelvalidate"
}

func (f *OELValidateFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Report whether a string is a syntactically valid Okta EL expression",
		MarkdownDescription: "Returns `true` if `expr` is a syntactically valid [Okta Expression Language](https://developer.okta.com/docs/reference/okta-expression-language/) expression, `false` otherwise. Unlike `oelformat`, it does not fail the plan on invalid input, so it suits a boolean check (for example in a `precondition` guarding a hand-written `okta_group_rule.expression_value`).\n\nValidation is syntax-only: it does not resolve attributes, check types, or evaluate the expression. It covers the full documented grammar (class calls, group builtins, receiver method calls, the Identity Engine method dialect, indexing, projection, Elvis, and `matches`). Backed by [okta-expression-parser](https://github.com/keeleysam/okta-expression-parser).",
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
		// Over the size guard, report not-valid rather than failing the plan, keeping the "does not fail the plan" contract absolute (an expression this large is not a real one).
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, false))
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, IsValid(expr)))
}
