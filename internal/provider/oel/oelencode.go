package oel

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = (*OELEncodeFunction)(nil)

type OELEncodeFunction struct{}

func NewOELEncodeFunction() function.Function { return &OELEncodeFunction{} }

func (f *OELEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "oelencode"
}

func (f *OELEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Build an Okta Expression Language string from an HCL data tree",
		MarkdownDescription: "Builds an [Okta Expression Language](https://developer.okta.com/docs/reference/okta-expression-language/) (OEL) string from a structured HCL value, so you assemble expressions from Terraform data (variables, `for` expressions, `merge`, `concat`) with no string templating and no manual quote escaping. The result is a canonical string suitable for `okta_group_rule.expression_value`, `okta_profile_mapping` mapping expressions, `okta_app_signon_policy_rule.custom_expression`, and Okta Identity Governance policy conditions.\n\n**Leaves.** Bare strings, numbers, booleans, and null are literals; a bare list is an OEL array literal (`{1, 2, 3}`). A reference (attribute path such as `user.department` or `appuser.email`) is the only marked leaf: `{ ident = \"user.department\" }`.\n\n**Operators** are OEL tokens or friendly aliases: `{ \"==\" = [a, b] }` or `{ eq = [a, b] }`; also `ne`/`lt`/`gt`/`le`/`ge`, `and`/`or` (n-ary; `&&`/`||` accepted and normalized to `AND`/`OR`), `not` (`{ not = a }`), `+` (n-ary concatenation), the ternary `{ cond = [test, ifTrue, ifFalse] }` (alias `{ \"?:\" = [...] }`), the Elvis operator `{ elvis = [value, default] }`, and the regex `{ matches = [subject, pattern] }` (`matches` is deprecated by Okta, kept for round-tripping existing expressions).\n\n**Calls** take three forms: a namespaced class method, `{ call = { class = \"String\", method = \"startsWith\", args = [ { ident = \"user.firstName\" }, \"prod-\" ] } }`; a bare function, `{ call = { function = \"isMemberOfAnyGroup\", args = [\"00g...\"] } }` (the `isMemberOf*` group builtins and any other bare function such as `substringBefore` or `getManagerUser`); and a receiver method call, `{ call = { target = { ident = \"user\" }, method = \"getInternalProperty\", args = [\"status\"] } }`, which also expresses the Identity Engine method dialect (`{ call = { target = { ident = \"user.profile.firstName\" }, method = \"toUpperCase\" } }`) and object-argument membership (`user.isMemberOf({...})`, with the object built via `map`).\n\n**Access and structure.** `{ select = { operand = <expr>, field = \"firstName\" } }` is a property access on a non-identifier receiver (a plain path should use `ident`); `{ index = { base = <expr>, index = <expr> } }` is `base[index]`; `{ project = { base = <expr>, expr = <expr> } }` is the collection projection `base.![expr]`; and `{ map = [ { key = \"group.profile.name\", value = \"X\" }, { key = \"operator\", value = \"EXACT\" } ] }` is an ordered object literal `{\"group.profile.name\": \"X\", \"operator\": \"EXACT\"}`.\n\nBacked by [okta-expression-parser](https://github.com/keeleysam/okta-expression-parser), which handles operator precedence, parenthesization, and quoting. The output is always parsed back before it is returned, so `oelencode` can never produce a syntactically invalid expression, and it is canonical (byte-identical to what `oelformat` produces from the same expression). An escape hatch `{ raw = \"<okta el>\" }` embeds a hand-written OEL fragment (it is parsed, and so validated). `!` and `not` are interchangeable, and logical operators are emitted in the `AND`/`OR` keyword form.",
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "expr",
				Description: "The expression tree, in the surface notation described above.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *OELEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var expr types.Dynamic

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &expr))
	if resp.Error != nil {
		return
	}

	if hasUnknown(expr) {
		// A value in the expression is unknown at plan time; return an unknown result so the plan proceeds and the value resolves at apply.
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.StringUnknown()))
		return
	}

	node, err := terraformToNode(expr.UnderlyingValue())
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to read expression: "+err.Error())
		return
	}

	out, err := Encode(node)
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
