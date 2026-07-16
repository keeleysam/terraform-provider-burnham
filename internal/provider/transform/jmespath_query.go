package transform

import (
	"context"
	_ "embed"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	jmespath "github.com/jmespath-community/go-jmespath"
)

//go:embed descriptions/jmespath_query.md
var jmespathQueryDescription string

var _ function.Function = (*JMESPathQueryFunction)(nil)

type JMESPathQueryFunction struct{}

func NewJMESPathQueryFunction() function.Function { return &JMESPathQueryFunction{} }

func (f *JMESPathQueryFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "jmespath_query"
}

func (f *JMESPathQueryFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Run a JMESPath query against a value",
		MarkdownDescription: jmespathQueryDescription,
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
	if unknownDynamicResultIfNeeded(ctx, resp, value.UnderlyingValue()) {
		return
	}

	data, err := terraformToJSON(value.UnderlyingValue())
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to convert value: "+err.Error())
		return
	}

	result, err := runJMESPath(data, expression)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(1, "JMESPath error: "+err.Error())
		return
	}

	tfVal, err := jsonToTerraform(result)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("failed to convert result: "+err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(tfVal)))
}

// runJMESPath is the pure core of the jmespath_query function: it evaluates an
// expression against a value in the JSON value space and returns the result.
func runJMESPath(data interface{}, expression string) (interface{}, error) {
	return jmespath.Search(expression, normalizeForJMESPath(data))
}

/*
normalizeForJMESPath recursively converts json.Number values (how terraformToJSON
emits every number, to preserve precision) into float64, the only numeric type the
go-jmespath interpreter understands. The interpreter does unchecked left.(float64)
assertions for the ordering operators (<, <=, >, >=) and arithmetic, and
reflect.DeepEqual for == / !=, so a json.Number silently compares as falsy, makes
arithmetic yield null, and trips the "array[number]" type check in max / sum / avg.
Converting up front makes numeric filters, arithmetic, and the numeric functions
behave as callers expect.
*/
func normalizeForJMESPath(v interface{}) interface{} {
	switch val := v.(type) {
	case json.Number:
		f, _ := val.Float64()
		return f
	case []interface{}:
		out := make([]interface{}, len(val))
		for i, e := range val {
			out[i] = normalizeForJMESPath(e)
		}
		return out
	case map[string]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, e := range val {
			out[k] = normalizeForJMESPath(e)
		}
		return out
	default:
		return v
	}
}
