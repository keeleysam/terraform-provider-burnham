package transform

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	jmespath "github.com/jmespath-community/go-jmespath"
)

var _ function.Function = (*JMESPathQueryFunction)(nil)

type JMESPathQueryFunction struct{}

func NewJMESPathQueryFunction() function.Function { return &JMESPathQueryFunction{} }

func (f *JMESPathQueryFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "jmespath_query"
}

func (f *JMESPathQueryFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Run a JMESPath query against a value",
		MarkdownDescription: "Evaluates a [JMESPath](https://jmespath.org/) expression against a Terraform value and returns the matching result. Useful for extracting fields from large nested structures (decoded API responses, manifests, configuration trees) without long chains of `try(local.x.foo[0].bar, null)`.\n\nThe expression follows the JMESPath specification — projections (`[*]`), filters (`[?key == 'value']`), pipes (`|`), functions (`length`, `sort_by`, `to_string`, …), and multi-select hashes (`{a: foo, b: bar}`) are all supported. Returns `null` when the expression matches nothing.\n\nBacked by [jmespath-community/go-jmespath](https://github.com/jmespath-community/go-jmespath), the actively-maintained fork of the canonical Go implementation.",
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "value",
				Description: "The value to query.",
			},
			function.StringParameter{
				Name:        "expression",
				Description: "A JMESPath expression.",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *JMESPathQueryFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value types.Dynamic
	var expression string

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value, &expression))
	if resp.Error != nil {
		return
	}

	data, err := terraformToJSON(value.UnderlyingValue())
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to convert value: "+err.Error()))
		return
	}

	result, err := jmespath.Search(expression, data)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("JMESPath error: "+err.Error()))
		return
	}

	tfVal, err := jsonToTerraform(result)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to convert result: "+err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(tfVal)))
}
