package transform

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/itchyny/gojq"
	"github.com/keeleysam/terraform-burnham/internal/provider/optionsutil"
)

var _ function.Function = (*JQFunction)(nil)

type JQFunction struct{}

func NewJQFunction() function.Function { return &JQFunction{} }

func (f *JQFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "jq"
}

func (f *JQFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Run a jq program against a value",
		MarkdownDescription: "Evaluates a [jq](https://jqlang.github.io/jq/) program against a Terraform value and returns the program's output stream as a list. jq is the most widely-used JSON query language; this is the expressive sibling of `jmespath_query` and `jsonpath_query`, with full support for pipes, `reduce`, `map`/`select`, string interpolation, object construction, and the rest of the jq language.\n\nBecause a jq program is a *stream* — `.[]` emits one result per element, `.a, .b` emits two — `jq` always returns a **list**, with one element per value the program produced. A program that yields a single value returns a one-element list; collapse it with `one(provider::burnham::jq(...))` or index the first element. A program that yields nothing returns an empty list.\n\nNamed bindings are passed through the optional `vars` object and referenced as jq variables: `provider::burnham::jq(local.data, \".items[] | select(.tier == $tier)\", { vars = { tier = \"prod\" } })` binds `$tier`.\n\n**Determinism caveat.** Most jq programs are pure functions of their input and produce the same output on every run. A few builtins are not: `now` and `localtime` read the wall clock (and host timezone), so any program deriving from them may produce different output on each plan or apply, and will churn the plan. Use them only when you intend that. The `env`/`$ENV` builtins return an empty object — this function does not expose the host process environment — and `input`/`inputs` error, because there is no secondary input stream. Numbers are handled with IEEE-754 `float64` precision for non-integers, matching the `jmespath_query` / `jsonpath_query` siblings; integers beyond 2⁵³ are preserved exactly.\n\nBacked by [itchyny/gojq](https://github.com/itchyny/gojq), a pure-Go jq implementation.",
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "value",
				Description: "The value to run the program against.",
			},
			function.StringParameter{
				Name:        "program",
				Description: "A jq program.",
			},
		},
		VariadicParameter: function.DynamicParameter{
			Name:               "options",
			Description:        "An optional object. Supported key: `vars` — an object of named bindings exposed to the program as jq variables (e.g. `{ vars = { tier = \"prod\" } }` binds `$tier`).",
			AllowNullValue:     false,
			AllowUnknownValues: false,
		},
		Return: function.DynamicReturn{},
	}
}

// jqOptions parses the optional options object, returning the `vars` map in the
// JSON value space (json.Number for numbers). Returns nil when no options or no
// vars were supplied.
func jqOptions(opts []types.Dynamic) (map[string]interface{}, *function.FuncError) {
	attrs, ferr := optionsutil.SingleOptionsObject(opts, `{ vars = { tier = "prod" } }`)
	if ferr != nil {
		return nil, ferr
	}
	var vars map[string]interface{}
	for k, val := range attrs {
		switch k {
		case "vars":
			obj, ok := val.(basetypes.ObjectValue)
			if !ok || obj.IsNull() || obj.IsUnknown() {
				return nil, function.NewArgumentFuncError(2, "options.vars must be an object")
			}
			conv, err := terraformToJSON(obj)
			if err != nil {
				return nil, function.NewArgumentFuncError(2, "options.vars: "+err.Error())
			}
			m, ok := conv.(map[string]interface{})
			if !ok {
				return nil, function.NewArgumentFuncError(2, "options.vars must be an object")
			}
			vars = m
		default:
			return nil, function.NewArgumentFuncError(2, fmt.Sprintf("unknown option key %q; the only supported key is vars", k))
		}
	}
	return vars, nil
}

func (f *JQFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value types.Dynamic
	var program string
	var optsArgs []types.Dynamic

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value, &program, &optsArgs))
	if resp.Error != nil {
		return
	}
	if unknownDynamicResultIfNeeded(ctx, resp, value.UnderlyingValue()) {
		return
	}
	for _, o := range optsArgs {
		if unknownDynamicResultIfNeeded(ctx, resp, o) {
			return
		}
	}

	vars, ferr := jqOptions(optsArgs)
	if ferr != nil {
		resp.Error = ferr
		return
	}

	input, err := terraformToJSON(value.UnderlyingValue())
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to convert value: "+err.Error())
		return
	}

	results, err := runJQ(input, program, vars)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(1, "jq error: "+err.Error())
		return
	}

	tfVal, err := jsonToTerraform(results)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("failed to convert result: "+err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(tfVal)))
}

// runJQ runs program against input and returns the output stream as a slice.
// input, vars, and the returned values are all in the JSON value space (numbers
// as json.Number); normalization to and from gojq's native number types
// (int / float64 / *big.Int) is confined to this function.
func runJQ(input interface{}, program string, vars map[string]interface{}) ([]interface{}, error) {
	query, err := gojq.Parse(program)
	if err != nil {
		return nil, err
	}

	// Variable names must be sorted so the positional values handed to Run line
	// up with the names handed to WithVariables.
	names := make([]string, 0, len(vars))
	for name := range vars {
		names = append(names, name)
	}
	sort.Strings(names)
	varNames := make([]string, len(names))
	varValues := make([]interface{}, len(names))
	for i, name := range names {
		varNames[i] = "$" + name
		varValues[i] = normalizeForGojq(vars[name])
	}

	code, err := gojq.Compile(query, gojq.WithVariables(varNames))
	if err != nil {
		return nil, err
	}

	results := []interface{}{}
	iter := code.RunWithContext(context.Background(), normalizeForGojq(input), varValues...)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if e, ok := v.(error); ok {
			return nil, e
		}
		norm, err := normalizeFromGojq(v)
		if err != nil {
			return nil, err
		}
		results = append(results, norm)
	}
	return results, nil
}

// normalizeForGojq converts a JSON-value-space value (json.Number for numbers)
// into the types gojq operates on: int / float64 / *big.Int. gojq rejects
// json.Number, so this conversion is mandatory on the way in.
func normalizeForGojq(v interface{}) interface{} {
	switch val := v.(type) {
	case json.Number:
		if i, err := val.Int64(); err == nil {
			return int(i)
		}
		if bi, ok := new(big.Int).SetString(string(val), 10); ok {
			return bi
		}
		f, _ := val.Float64()
		return f
	case []interface{}:
		out := make([]interface{}, len(val))
		for i, e := range val {
			out[i] = normalizeForGojq(e)
		}
		return out
	case map[string]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, e := range val {
			out[k] = normalizeForGojq(e)
		}
		return out
	default:
		return v
	}
}

// normalizeFromGojq converts gojq output back into the JSON value space, mapping
// every number flavour gojq emits (int / float64 / *big.Int) to json.Number so
// the whole transform package stays consistently in the json.Number space that
// jsonToTerraform consumes. A jq program can build a result nested arbitrarily
// deep (e.g. reduce over a huge range), so recursion is bounded by
// transformMaxDepth to return an error rather than overflow the goroutine stack.
func normalizeFromGojq(v interface{}) (interface{}, error) {
	return normalizeFromGojqImpl(v, 0)
}

func normalizeFromGojqImpl(v interface{}, depth int) (interface{}, error) {
	if depth >= transformMaxDepth {
		return nil, fmt.Errorf("result exceeds maximum supported nesting depth of %d", transformMaxDepth)
	}
	switch val := v.(type) {
	case int:
		return json.Number(strconv.Itoa(val)), nil
	case int64:
		return json.Number(strconv.FormatInt(val, 10)), nil
	case *big.Int:
		return json.Number(val.String()), nil
	case float64:
		return json.Number(strconv.FormatFloat(val, 'f', -1, 64)), nil
	case []interface{}:
		out := make([]interface{}, len(val))
		for i, e := range val {
			conv, err := normalizeFromGojqImpl(e, depth+1)
			if err != nil {
				return nil, err
			}
			out[i] = conv
		}
		return out, nil
	case map[string]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, e := range val {
			conv, err := normalizeFromGojqImpl(e, depth+1)
			if err != nil {
				return nil, err
			}
			out[k] = conv
		}
		return out, nil
	default:
		return v, nil
	}
}
