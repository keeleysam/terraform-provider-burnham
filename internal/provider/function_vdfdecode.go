package provider

import (
	"context"
	"strings"

	"github.com/andygrunwald/vdf"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = (*VDFDecodeFunction)(nil)

type VDFDecodeFunction struct{}

func NewVDFDecodeFunction() function.Function {
	return &VDFDecodeFunction{}
}

func (f *VDFDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "vdfdecode"
}

func (f *VDFDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Parse a Valve Data Format (VDF) string into a Terraform value",
		Description: "Decodes a VDF string (used by Steam/Source engine) into a Terraform object. " +
			"VDF is a nested key-value format with only strings and objects — all leaf values are strings. " +
			"Comments (//) are stripped during parsing.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "input",
				Description: "A VDF string to parse.",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *VDFDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))
	if resp.Error != nil {
		return
	}

	parser := vdf.NewParser(strings.NewReader(input))
	goVal, err := parser.Parse()
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to parse VDF: "+err.Error()))
		return
	}

	tfVal, err := goToTerraformValue(goVal)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to convert VDF value: "+err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(tfVal)))
}
