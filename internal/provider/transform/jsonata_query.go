package transform

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/recolabs/gnata"
)

// jsonataMaxInputBytes bounds the length of a JSONata expression. A JSONata
// program is small in every realistic use, so an expression past this size is a
// mistake or an attempt to make the parser do pathological work at plan time.
// 16 MiB matches the other expression-language functions in this repository.
const jsonataMaxInputBytes = 16 << 20

//go:embed descriptions/jsonata_query.md
var jsonataQueryDescription string

var _ function.Function = (*JSONataQueryFunction)(nil)

type JSONataQueryFunction struct{}

func NewJSONataQueryFunction() function.Function { return &JSONataQueryFunction{} }

func (f *JSONataQueryFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "jsonata_query"
}

func (f *JSONataQueryFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Run a JSONata expression against a value",
		MarkdownDescription: jsonataQueryDescription,
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "value",
				Description: "The value to query.",
			},
			function.StringParameter{
				Name:        "expression",
				Description: "A JSONata expression.",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *JSONataQueryFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value types.Dynamic
	var expression string

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value, &expression))
	if resp.Error != nil {
		return
	}
	if unknownDynamicResultIfNeeded(ctx, resp, value.UnderlyingValue()) {
		return
	}
	if len(expression) > jsonataMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(1, fmt.Sprintf("expression exceeds maximum supported length of %d bytes", jsonataMaxInputBytes))
		return
	}

	data, err := terraformToJSON(value.UnderlyingValue())
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to convert value: "+err.Error())
		return
	}

	result, err := runJSONata(ctx, data, expression)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(1, "JSONata error: "+err.Error())
		return
	}

	tfVal, err := jsonToTerraform(result)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("failed to convert result: "+err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(tfVal)))
}

/*
jsonataDeterministicEnv is a gnata environment that overrides JSONata's three
non-deterministic builtins ($now, $millis, $random) so any expression that calls
them fails instead of producing plan-churning output. burnham is a pure provider:
plan output must equal apply output, so these builtins have no valid use in
jsonata_query. The environment is created once and is safe for concurrent reuse
(gnata documents NewCustomEnv's result as goroutine-safe for reads).
*/
var jsonataDeterministicEnv = gnata.NewCustomEnv(map[string]gnata.CustomFunc{
	"now":    blockNonDeterministicJSONata("now"),
	"millis": blockNonDeterministicJSONata("millis"),
	"random": blockNonDeterministicJSONata("random"),
})

func blockNonDeterministicJSONata(name string) gnata.CustomFunc {
	return func(_ []any, _ any) (any, error) {
		return nil, fmt.Errorf("$%s is disabled: burnham is a pure provider, so the non-deterministic JSONata builtins ($now, $millis, $random) cannot be used in jsonata_query", name)
	}
}

/*
runJSONata is the pure core of jsonata_query: it evaluates expression against
data (both in the JSON value space, json.Number for numbers) and returns the
result in the same space.

The input is round-tripped through json.Marshal and gnata.DecodeJSON before
evaluation. json.Marshal sorts object keys, and DecodeJSON preserves that order
while keeping numbers as json.Number, so order-sensitive builtins ($keys, $each,
$spread) return values in a stable order and never churn the plan. Evaluation
uses the deterministic environment so $now / $millis / $random are rejected.
gnata.NormalizeValue converts gnata's internal OrderedMap and null sentinel back
into plain map[string]interface{} / nil that jsonToTerraform understands.
*/
func runJSONata(ctx context.Context, data interface{}, expression string) (interface{}, error) {
	expr, err := gnata.Compile(expression)
	if err != nil {
		return nil, err
	}

	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	decoded, err := gnata.DecodeJSON(raw)
	if err != nil {
		return nil, err
	}

	result, err := expr.EvalWithCustomFuncs(ctx, decoded, jsonataDeterministicEnv)
	if err != nil {
		return nil, err
	}
	return gnata.NormalizeValue(result), nil
}
