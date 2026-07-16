package transform

import (
	"context"
	_ "embed"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/theory/jsonpath"
)

//go:embed descriptions/jsonpath_query.md
var jsonpathQueryDescription string

var _ function.Function = (*JSONPathQueryFunction)(nil)

type JSONPathQueryFunction struct{}

func NewJSONPathQueryFunction() function.Function { return &JSONPathQueryFunction{} }

func (f *JSONPathQueryFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "jsonpath_query"
}

func (f *JSONPathQueryFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Run a JSONPath query (RFC 9535) against a value",
		MarkdownDescription: jsonpathQueryDescription,
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
	if unknownDynamicResultIfNeeded(ctx, resp, value.UnderlyingValue()) {
		return
	}

	path, err := jsonpath.Parse(expression)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(1, "invalid JSONPath expression: "+err.Error())
		return
	}

	data, err := terraformToJSON(value.UnderlyingValue())
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to convert value: "+err.Error())
		return
	}

	nodes := path.Select(data)
	results := make([]interface{}, len(nodes))
	for i, n := range nodes {
		results[i] = n
	}

	tfVal, err := jsonToTerraform(results)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("failed to convert result: "+err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(tfVal)))
}
