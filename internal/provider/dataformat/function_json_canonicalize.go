package dataformat

import (
	_ "embed"

	"context"
	"encoding/json"
	"math/big"

	"github.com/gowebpki/jcs"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = (*JSONCanonicalizeFunction)(nil)

//go:embed descriptions/json_canonicalize.md
var jsonCanonicalizeDescription string

type JSONCanonicalizeFunction struct{}

func NewJSONCanonicalizeFunction() function.Function { return &JSONCanonicalizeFunction{} }

func (f *JSONCanonicalizeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "json_canonicalize"
}

func (f *JSONCanonicalizeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Serialize a value as RFC 8785 canonical JSON (JCS)",
		MarkdownDescription: jsonCanonicalizeDescription,
		Parameters: []function.Parameter{
			function.DynamicParameter{Name: "value", Description: "The value to canonicalize."},
		},
		Return: function.StringReturn{},
	}
}

func (f *JSONCanonicalizeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value))
	if resp.Error != nil {
		return
	}
	// Pass the Dynamic wrapper (not its underlying value) so a fully-unknown argument, whose UnderlyingValue() is nil, is still detected alongside nested unknowns.
	if unknownStringResultIfNeeded(ctx, resp, value, nil) {
		return
	}

	goVal, err := terraformValueToGo(value.UnderlyingValue(), false)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to convert value: "+err.Error())
		return
	}

	// Marshal to JSON first, then canonicalize. Numbers are carried as json.Number tokens so json.Marshal emits them as bare number literals (never quoted strings, which is what a bare *big.Int would produce through encoding.TextMarshaler); RFC 8785's serializer then reformats every number under the I-JSON / ES6 double rules.
	raw, err := json.Marshal(goValueForCanonicalize(goVal))
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("failed to encode JSON: "+err.Error()))
		return
	}
	canon, err := jcs.Transform(raw)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(0, "failed to canonicalize: "+err.Error()))
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, string(canon)))
}

// goValueForCanonicalize converts a terraformValueToGo result into a shape json.Marshal renders with bare number tokens. int64 and float64 already marshal as numbers; *big.Int / big.Int (produced for integers beyond int64) would otherwise marshal as quoted strings via encoding.TextMarshaler, so they become json.Number instead.
func goValueForCanonicalize(v interface{}) interface{} {
	switch val := v.(type) {
	case *big.Int:
		return json.Number(val.String())
	case big.Int:
		return json.Number(val.String())
	case []interface{}:
		out := make([]interface{}, len(val))
		for i, item := range val {
			out[i] = goValueForCanonicalize(item)
		}
		return out
	case map[string]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, item := range val {
			out[k] = goValueForCanonicalize(item)
		}
		return out
	default:
		return val
	}
}
