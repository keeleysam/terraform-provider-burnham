package dataformat

import (
	_ "embed"

	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = (*NDJSONDecodeFunction)(nil)

//go:embed descriptions/ndjsondecode.md
var ndjsondecodeDescription string

type NDJSONDecodeFunction struct{}

func NewNDJSONDecodeFunction() function.Function { return &NDJSONDecodeFunction{} }

func (f *NDJSONDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "ndjsondecode"
}

func (f *NDJSONDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Parse NDJSON (newline-delimited JSON) into a list of values",
		MarkdownDescription: ndjsondecodeDescription,
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "input",
				Description: "An NDJSON / JSON Lines string.",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *NDJSONDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))
	if resp.Error != nil {
		return
	}

	if len(input) > dataformatMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("input exceeds maximum supported length of %d bytes", dataformatMaxInputBytes))
		return
	}
	var values []interface{}
	dec := json.NewDecoder(strings.NewReader(input))
	dec.UseNumber()
	lineNum := 0
	for {
		// Skip blank lines / trailing whitespace by checking if more tokens remain.
		if !dec.More() {
			break
		}
		lineNum++
		var v interface{}
		if err := dec.Decode(&v); err != nil {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(fmt.Sprintf("Failed to decode NDJSON record %d: %s", lineNum, err.Error())))
			return
		}
		values = append(values, v)
	}

	if len(values) == 0 {
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(types.TupleValueMust([]attr.Type{}, []attr.Value{}))))
		return
	}

	tfVal, err := goSliceToTuple(values)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to convert value: "+err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(tfVal)))
}

var _ function.Function = (*NDJSONEncodeFunction)(nil)

//go:embed descriptions/ndjsonencode.md
var ndjsonencodeDescription string

type NDJSONEncodeFunction struct{}

func NewNDJSONEncodeFunction() function.Function { return &NDJSONEncodeFunction{} }

func (f *NDJSONEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "ndjsonencode"
}

func (f *NDJSONEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Encode a list of values as NDJSON",
		MarkdownDescription: ndjsonencodeDescription,
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "value",
				Description: "A list (tuple) of values to encode, one per line.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *NDJSONEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value))
	if resp.Error != nil {
		return
	}
	if unknownStringResultIfNeeded(ctx, resp, value.UnderlyingValue(), nil) {
		return
	}

	goVal, err := terraformValueToGo(value.UnderlyingValue(), false)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to convert value: "+err.Error())
		return
	}

	slice, ok := goVal.([]interface{})
	if !ok {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(fmt.Sprintf("ndjsonencode requires a list, got %T", goVal)))
		return
	}

	var buf bytes.Buffer
	for i, item := range slice {
		prepared := goValueForJSONEncode(item)
		b, err := json.Marshal(prepared)
		if err != nil {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(fmt.Sprintf("Failed to encode element %d: %s", i, err.Error())))
			return
		}
		buf.Write(b)
		buf.WriteByte('\n')
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, buf.String()))
}
