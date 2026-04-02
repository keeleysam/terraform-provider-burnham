package provider

import (
	"context"
	"encoding/base64"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = (*PlistDataFunction)(nil)

type PlistDataFunction struct{}

func NewPlistDataFunction() function.Function {
	return &PlistDataFunction{}
}

func (f *PlistDataFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "plistdata"
}

func (f *PlistDataFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Create a plist data value from a base64-encoded string",
		Description: "Returns a tagged object representing an NSData plist value. When passed to plistencode, this produces a <data> element. The same tagged object format is returned by plistdecode for <data> elements, enabling seamless round-trips. Use with filebase64() to embed binary data such as certificates.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "base64",
				Description: "A base64-encoded string, e.g. from filebase64().",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *PlistDataFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))
	if resp.Error != nil {
		return
	}

	// Validate the input is valid base64.
	_, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(
			"Invalid base64 input: "+err.Error()))
		return
	}

	obj, err := makePlistTaggedObject(plistTypeData, input)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(obj)))
}
