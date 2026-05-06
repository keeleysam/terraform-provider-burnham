package dataformat

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = (*INIEncodeFunction)(nil)

type INIEncodeFunction struct{}

func NewINIEncodeFunction() function.Function {
	return &INIEncodeFunction{}
}

func (f *INIEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "iniencode"
}

func (f *INIEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Encode a value as an INI file",
		Description: "Encodes a Terraform object as an INI string. The input must be a map of section names to maps of key-value pairs. The empty string key (\"\") renders as global keys before any section header. All values are converted to strings.",
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "value",
				Description: "An object of {section_name = {key = value}} to encode as INI.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *INIEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value types.Dynamic

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value))
	if resp.Error != nil {
		return
	}

	obj, ok := value.UnderlyingValue().(types.Object)
	if !ok {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Value must be an object with section names as keys."))
		return
	}

	result := renderINI(obj.Attributes())

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, result))
}
