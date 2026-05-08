package dataformat

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
		Summary: "Encode a value as pretty-printed JSON",
		MarkdownDescription: "Encodes a Terraform value as a pretty-printed JSON string with newlines and indentation. Unlike Terraform's built-in `jsonencode`, which produces a single compact line, this function returns output that's reviewable in pull requests and diff-friendly when written to a file.\n\nPass an optional `options` object with an `indent` key (string) to override the default tab indentation — e.g. `{ indent = \"  \" }` for two-space indent. Object keys are sorted alphabetically; whole numbers render without a decimal point.\n\n**Common uses:** rendering IAM policies, OpenAPI specs, or any structured JSON document that gets reviewed in PRs or written to disk via `local_file`.",
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "value",
				Description: "The value to encode as JSON.",
			},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: "An optional options object. Supported keys: \"indent\" (string) — indentation string, default \"\\t\". Pass at most one.",
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

	indent := "\t"
	if len(optsArgs) == 1 {
		parsed, err := parseOptionsIndent(optsArgs[0])
		if err != nil {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
			return
		}
		if parsed != "" {
			indent = parsed
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

	result, err := json.MarshalIndent(prepared, "", indent)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to encode JSON: "+err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, string(result)))
}

