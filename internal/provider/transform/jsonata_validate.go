package transform

import (
	"context"
	_ "embed"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/recolabs/gnata"
)

//go:embed descriptions/jsonata_validate.md
var jsonataValidateDescription string

var _ function.Function = (*JSONataValidateFunction)(nil)

type JSONataValidateFunction struct{}

func NewJSONataValidateFunction() function.Function { return &JSONataValidateFunction{} }

func (f *JSONataValidateFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "jsonata_validate"
}

func (f *JSONataValidateFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Report whether a string is a syntactically valid JSONata expression",
		MarkdownDescription: jsonataValidateDescription,
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "expression",
				Description: "A JSONata expression string to check.",
			},
		},
		Return: function.BoolReturn{},
	}
}

func (f *JSONataValidateFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var expression string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &expression))
	if resp.Error != nil {
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.BoolValue(isValidJSONata(expression))))
}

/*
isValidJSONata reports whether expression parses as JSONata. It is syntax-only:
an expression that references undefined fields or the non-deterministic builtins
($now, $millis, $random) is still syntactically valid, so it returns true (only
jsonata_query rejects those at evaluation time). An oversized expression returns
false rather than an error, so the validate function never fails the plan and
stays usable inside a precondition.
*/
func isValidJSONata(expression string) bool {
	if len(expression) > jsonataMaxInputBytes {
		return false
	}
	_, err := gnata.Compile(expression)
	return err == nil
}
