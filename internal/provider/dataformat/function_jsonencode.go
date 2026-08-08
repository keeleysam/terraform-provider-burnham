package dataformat

import (
	_ "embed"

	"bytes"
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/tidwall/pretty"
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
			Description: "An optional options object. Supported keys: \"indent\" (string, default \"\\t\"), \"width\" (number, default 0 = one member or element per line) and \"escape_html\" (bool, default false). Pass at most one.",
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
	width := 0
	if len(optsArgs) == 1 {
		obj, ok := optsArgs[0].UnderlyingValue().(basetypes.ObjectValue)
		if !ok {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(1, "options must be an object"))
			return
		}
		attrs := obj.Attributes()
		if err := validateOptionKeys(attrs, "indent", "escape_html", "width"); err != nil {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(1, err.Error()))
			return
		}
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
		w, wPresent, err := getIntOption(attrs, "width")
		if err != nil {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
			return
		}
		if wPresent {
			if w < 0 {
				resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(1, "\"width\" must not be negative"))
				return
			}
			width = w
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

	// One encode pass either way; the only difference is who inserts the line
	// breaks. json.MarshalIndent always HTML-escapes, so an Encoder is the only
	// way to turn that off. When width is set the encoder emits a single line and
	// tidwall decides the breaks; otherwise SetIndent gives one member or element
	// per line.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(escapeHTML)
	if width == 0 {
		enc.SetIndent("", indent)
	}
	if err := enc.Encode(prepared); err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to encode JSON: "+err.Error()))
		return
	}
	// Encode appends a trailing newline that MarshalIndent does not.
	result := bytes.TrimRight(buf.Bytes(), "\n")

	if width > 0 {
		// tidwall's Width applies to arrays of scalars only and never to objects,
		// so an object is always alone on its line and the "}, {" and "[{" shapes
		// cannot occur. Keys are already sorted, since prepared is a map and
		// encoding/json sorts map keys, so SortKeys would only redo that work.
		result = bytes.TrimRight(pretty.PrettyOptions(result, &pretty.Options{Width: width, Indent: indent}), "\n")
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, string(result)))
}
