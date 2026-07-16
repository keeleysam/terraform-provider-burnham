package dataformat

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var _ function.Function = (*JSONEncodeFunction)(nil)

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
		MarkdownDescription: "Encodes a Terraform value as a pretty-printed JSON string with newlines and indentation. Unlike Terraform's built-in `jsonencode`, which produces a single compact line, this function returns output that's reviewable in pull requests and diff-friendly when written to a file.\n\nThe optional `options` object supports:\n\n- `indent` (string): override the default tab indentation, e.g. `{ indent = \"  \" }` for two-space indent.\n- `escape_html` (bool, default `false`): when `false`, `<`, `>` and `&` are written literally, which is what you want for human-reviewed output. Terraform's built-in `jsonencode` (and Go's encoder) escape them to `\\u003c` / `\\u003e` / `\\u0026`; set this to `true` to match that legacy behavior, e.g. when embedding JSON in an HTML `<script>` context.\n\nObject keys are sorted alphabetically; whole numbers render without a decimal point.\n\n**Common uses:** rendering IAM policies, OpenAPI specs, or any structured JSON document that gets reviewed in PRs or written to disk via `local_file`.",
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
