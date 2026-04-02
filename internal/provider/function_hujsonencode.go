package provider

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tailscale/hujson"
)

var _ function.Function = (*HuJSONEncodeFunction)(nil)

type HuJSONEncodeFunction struct{}

func NewHuJSONEncodeFunction() function.Function {
	return &HuJSONEncodeFunction{}
}

func (f *HuJSONEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "hujsonencode"
}

func (f *HuJSONEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Encode a value as a HuJSON (JWCC) string",
		Description: "Encodes a Terraform value as a HuJSON string with trailing commas and pretty-printed formatting. Default indentation is a tab character; pass an optional indent string to override.",
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "value",
				Description: "The value to encode as HuJSON.",
			},
		},
		VariadicParameter: function.StringParameter{
			Name:        "indent",
			Description: "The string to use for each indentation level. Defaults to a tab character. Pass at most one value.",
		},
		Return: function.StringReturn{},
	}
}

func (f *HuJSONEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value types.Dynamic
	var indentArgs []string

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value, &indentArgs))
	if resp.Error != nil {
		return
	}

	indent := "\t"
	if len(indentArgs) == 1 {
		indent = indentArgs[0]
	} else if len(indentArgs) > 1 {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(1, "At most one indent argument may be provided."))
		return
	}

	goVal, err := terraformValueToGo(value.UnderlyingValue(), false)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to convert value: "+err.Error()))
		return
	}

	prepared := goValueForJSONEncode(goVal)

	jsonBytes, err := json.Marshal(prepared)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to marshal JSON: "+err.Error()))
		return
	}

	// Make the JSON non-standard so hujson.Format() will add trailing commas.
	// We prepend a line comment which makes IsStandard() return false.
	hujsonBytes := append([]byte("//\n"), jsonBytes...)

	ast, err := hujson.Parse(hujsonBytes)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to parse as HuJSON: "+err.Error()))
		return
	}

	ast.Format()

	// Remove the injected comment from the output.
	ast.BeforeExtra = nil

	result := string(ast.Pack())

	// hujson.Format() always uses tabs. If a different indent was requested,
	// replace the leading tabs on each line.
	if indent != "\t" {
		lines := strings.Split(result, "\n")
		for i, line := range lines {
			trimmed := strings.TrimLeft(line, "\t")
			tabCount := len(line) - len(trimmed)
			if tabCount > 0 {
				lines[i] = strings.Repeat(indent, tabCount) + trimmed
			}
		}
		result = strings.Join(lines, "\n")
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, result))
}
