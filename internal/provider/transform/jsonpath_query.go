package transform

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/theory/jsonpath"
)

var _ function.Function = (*JSONPathQueryFunction)(nil)

type JSONPathQueryFunction struct{}

func NewJSONPathQueryFunction() function.Function { return &JSONPathQueryFunction{} }

func (f *JSONPathQueryFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "jsonpath_query"
}

func (f *JSONPathQueryFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Run a JSONPath query (RFC 9535) against a value",
		MarkdownDescription: "Evaluates an [RFC 9535](https://www.rfc-editor.org/rfc/rfc9535.html) JSONPath expression against a Terraform value and returns the list of matching nodes. Useful for extracting subsets of nested structures using the standardized JSONPath grammar.\n\nThe expression must conform to RFC 9535 — the IETF standardized JSONPath. Common selectors include the root identifier (`$`), name selectors (`$.store.book`), wildcard (`$..*`), descendant segments (`$..price`), array slices (`$[0:5]`), and filters (`$[?@.price < 10]`).\n\nReturns a list of matching values. An expression that matches nothing returns an empty list. To collapse single-match queries to a scalar, use `one(provider::burnham::jsonpath_query(...))` or index the first element.\n\nBacked by [theory/jsonpath](https://github.com/theory/jsonpath), an RFC 9535 conforming implementation.",
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "value",
				Description: "The value to query.",
			},
			function.StringParameter{
				Name:        "expression",
				Description: "An RFC 9535 JSONPath expression.",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *JSONPathQueryFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value types.Dynamic
	var expression string

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value, &expression))
	if resp.Error != nil {
		return
	}

	path, err := jsonpath.Parse(expression)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Invalid JSONPath expression: "+err.Error()))
		return
	}

	data, err := terraformToJSON(value.UnderlyingValue())
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to convert value: "+err.Error()))
		return
	}

	nodes := path.Select(data)
	results := make([]interface{}, len(nodes))
	for i, n := range nodes {
		results[i] = n
	}

	tfVal, err := jsonToTerraform(results)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to convert result: "+err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(tfVal)))
}
