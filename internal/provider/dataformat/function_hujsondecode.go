package dataformat

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tailscale/hujson"
)

var _ function.Function = (*HuJSONDecodeFunction)(nil)

type HuJSONDecodeFunction struct{}

func NewHuJSONDecodeFunction() function.Function {
	return &HuJSONDecodeFunction{}
}

func (f *HuJSONDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "hujsondecode"
}

func (f *HuJSONDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Parse a HuJSON (JWCC) string into a Terraform value",
		Description: "Decodes a HuJSON string (JSON with comments and trailing commas) into a Terraform value. Comments are stripped during parsing.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "input",
				Description: "A HuJSON string to decode.",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *HuJSONDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))
	if resp.Error != nil {
		return
	}

	// Standardize strips comments and trailing commas, producing valid JSON.
	standardized, err := hujson.Standardize([]byte(input))
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Invalid HuJSON: "+err.Error()))
		return
	}

	// Unmarshal to Go types using json.Number for precision.
	d := json.NewDecoder(bytes.NewReader(standardized))
	d.UseNumber()
	var goVal interface{}
	if err := d.Decode(&goVal); err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to decode JSON: "+err.Error()))
		return
	}

	tfVal, err := goToTerraformValue(goVal)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to convert value: "+err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(tfVal)))
}
