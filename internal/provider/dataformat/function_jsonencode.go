package dataformat

import (
	_ "embed"

	"bytes"
	"context"
	"encoding/json"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var _ function.Function = (*JSONEncodeFunction)(nil)

//go:embed descriptions/jsonencode.md
var jsonencodeDescription string

type JSONEncodeFunction struct{}

func NewJSONEncodeFunction() function.Function {
	return &JSONEncodeFunction{}
}

func (f *JSONEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "jsonencode"
}

func (f *JSONEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Encode a value as pretty-printed JSON",
		MarkdownDescription: jsonencodeDescription,
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "value",
				Description: "The value to encode as JSON.",
			},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: "An optional options object. Supported keys: \"indent\" (string, default \"\\t\") and \"escape_html\" (bool, default false). Pass at most one.",
		},
		Return: function.StringReturn{},
	}
}

func (f *JSONEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value types.Dynamic
	var optsArgs []types.Dynamic

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value, &optsArgs))
	if resp.Error != nil {
		return
	}
	if unknownStringResultIfNeeded(ctx, resp, value.UnderlyingValue(), optsArgs) {
		return
	}

	indent := "\t"
	escapeHTML := false
	if len(optsArgs) == 1 {
		obj, ok := optsArgs[0].UnderlyingValue().(basetypes.ObjectValue)
		if !ok {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(1, "options must be an object"))
			return
		}
		attrs := obj.Attributes()
		parsed, err := getStringOption(attrs, "indent")
		if err != nil {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
			return
		}
		if parsed != "" {
			indent = parsed
		}
		esc, present, err := getBoolOption(attrs, "escape_html")
		if err != nil {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
			return
		}
		if present {
			escapeHTML = esc
		}
	} else if len(optsArgs) > 1 {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(1, "At most one options argument may be provided."))
		return
	}

	goVal, err := terraformValueToGo(value.UnderlyingValue(), false)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to convert value: "+err.Error())
		return
	}

	prepared := goValueForJSONEncode(goVal)

	// json.MarshalIndent always HTML-escapes; an Encoder is the only way to turn
	// that off. Encode appends a trailing newline that MarshalIndent does not,
	// so trim it to keep the output stable.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(escapeHTML)
	enc.SetIndent("", indent)
	if err := enc.Encode(prepared); err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to encode JSON: "+err.Error()))
		return
	}
	result := strings.TrimRight(buf.String(), "\n")

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, result))
}
