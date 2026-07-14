package cel

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/keeleysam/terraform-burnham/internal/provider/optionsutil"
)

var _ function.Function = (*CELEvaluateFunction)(nil)

type CELEvaluateFunction struct{}

func NewCELEvaluateFunction() function.Function { return &CELEvaluateFunction{} }

func (f *CELEvaluateFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "celevaluate"
}

func (f *CELEvaluateFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Evaluate a standard CEL expression at plan time",
		MarkdownDescription: "Compiles and evaluates a [CEL](https://cel.dev) expression against variable bindings and returns the result as a Terraform value. Useful for testing the logic of an expression you built with `celencode`, and for computing or validating values inside a plan.\n\nThis evaluates **standard CEL only**: cel-go's standard library plus its extension libraries (strings, math, lists, sets, encoders, bindings, two-variable comprehensions, regex, network) and optional types. Dialect-specific functions provided by a downstream host (GCP's `inIpRange`, Kubernetes' `quantity`/`authorizer`, and the like) are **not** available and will fail to compile, since this provider does not implement them. Every variable referenced by the expression must be supplied in `vars`; an undeclared variable or function is a compile error.\n\nEvaluation is deterministic (CEL has no wall-clock or randomness), so results are stable across plan and apply. Variables are declared dynamically, so no type annotations are needed. Result values map to Terraform as follows (each overridable in options): a timestamp becomes an RFC 3339 string, a duration a seconds string like `\"5400s\"`, bytes a base64 string, and a map with non-string keys is an error (Terraform objects require string keys). An absent optional (`optional.none()`) becomes null, indistinguishable from a CEL null result.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "expr",
				Description: "A standard CEL expression to evaluate.",
			},
		},
		VariadicParameter: function.DynamicParameter{
			Name:               "options",
			Description:        "An optional object. Keys: `vars` (object of variable bindings referenced by the expression); `cost_limit` (number, default cel-go's unbounded default) to bound evaluation cost; `timestamp_format` (`rfc3339` default, or `unix`); `duration_format` (`string` default e.g. `\"5400s\"`, `go` e.g. `\"1h30m0s\"`, or `seconds` number); `bytes_format` (`base64` default, or `hex`).",
			AllowNullValue:     false,
			AllowUnknownValues: false,
		},
		Return: function.DynamicReturn{},
	}
}

func (f *CELEvaluateFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var expr string
	var optsArgs []types.Dynamic

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &expr, &optsArgs))
	if resp.Error != nil {
		return
	}

	if len(expr) > celMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("expression exceeds maximum supported length of %d bytes", celMaxInputBytes))
		return
	}
	if optionsHaveUnknown(optsArgs) {
		// A variable binding or option is unknown at plan time; return an unknown result so the plan resolves at apply.
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicUnknown()))
		return
	}

	vars, opts, ferr := celEvalOptions(optsArgs)
	if ferr != nil {
		resp.Error = ferr
		return
	}

	result, err := Eval(expr, vars, opts)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	value, err := nodeToAttr(result)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(value)))
}

// celEvalOptions parses the optional options object for celevaluate: vars plus the result-format and cost-limit knobs.
// The options object is at parameter index 1; the caller checks for unknown values before calling this.
func celEvalOptions(optsArgs []types.Dynamic) (vars map[string]any, opts evalOptions, ferr *function.FuncError) {
	opts = defaultEvalOptions()
	attrs, ferr := optionsutil.SingleOptionsObject(optsArgs, `{ vars = { x = 1 } }`)
	if ferr != nil {
		return nil, opts, ferr
	}
	for k, v := range attrs {
		switch k {
		case "vars":
			obj, ok := v.(basetypes.ObjectValue)
			if !ok || obj.IsNull() || obj.IsUnknown() {
				return nil, opts, function.NewArgumentFuncError(1, "options.vars must be an object")
			}
			node, err := terraformToNode(obj)
			if err != nil {
				return nil, opts, function.NewArgumentFuncError(1, "options.vars: "+err.Error())
			}
			m, ok := normalizeForEval(node).(map[string]any)
			if !ok {
				return nil, opts, function.NewArgumentFuncError(1, "options.vars must be an object")
			}
			vars = m
		case "cost_limit":
			n, err := optionsutil.NumberAttrToInt(v)
			if err != nil {
				return nil, opts, function.NewArgumentFuncError(1, "options.cost_limit: "+err.Error())
			}
			if n < 0 {
				return nil, opts, function.NewArgumentFuncError(1, "options.cost_limit must be non-negative")
			}
			opts.costLimit = uint64(n)
		case "timestamp_format":
			s, ferr := stringOption(v, "timestamp_format", "rfc3339", "unix")
			if ferr != nil {
				return nil, opts, ferr
			}
			opts.tsFormat = s
		case "duration_format":
			s, ferr := stringOption(v, "duration_format", "string", "go", "seconds")
			if ferr != nil {
				return nil, opts, ferr
			}
			opts.durFormat = s
		case "bytes_format":
			s, ferr := stringOption(v, "bytes_format", "base64", "hex")
			if ferr != nil {
				return nil, opts, ferr
			}
			opts.bytesFormat = s
		default:
			return nil, opts, function.NewArgumentFuncError(1, fmt.Sprintf("unknown option key %q; supported: vars, cost_limit, timestamp_format, duration_format, bytes_format", k))
		}
	}
	return vars, opts, nil
}

func stringOption(v attr.Value, key string, allowed ...string) (string, *function.FuncError) {
	s, ok := v.(basetypes.StringValue)
	if !ok || s.IsNull() || s.IsUnknown() {
		return "", function.NewArgumentFuncError(1, fmt.Sprintf("options.%s must be a string", key))
	}
	val := s.ValueString()
	for _, a := range allowed {
		if val == a {
			return val, nil
		}
	}
	return "", function.NewArgumentFuncError(1, fmt.Sprintf("options.%s must be one of %v; got %q", key, allowed, val))
}

// normalizeForEval converts the json.Number values in a decoded tree into concrete int64/float64 so cel-go's activation adapter accepts them.
func normalizeForEval(node any) any {
	switch v := node.(type) {
	case json.Number:
		if bf, _, err := big.ParseFloat(v.String(), 10, 512, big.ToNearestEven); err == nil && bf.IsInt() {
			if i, acc := bf.Int64(); acc == big.Exact {
				return i
			}
		}
		f, _ := v.Float64()
		return f
	case []any:
		out := make([]any, len(v))
		for i, el := range v {
			out[i] = normalizeForEval(el)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, el := range v {
			out[k] = normalizeForEval(el)
		}
		return out
	default:
		return node
	}
}
