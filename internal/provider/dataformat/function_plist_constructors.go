package dataformat

import (
	"context"
	"encoding/base64"
	"math/big"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = (*PlistDateFunction)(nil)

type PlistDateFunction struct{}

func NewPlistDateFunction() function.Function {
	return &PlistDateFunction{}
}

func (f *PlistDateFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "plistdate"
}

func (f *PlistDateFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Create a plist date value from an RFC 3339 timestamp",
		MarkdownDescription: "Returns a tagged object representing an NSDate plist value. When passed to plistencode, this produces a <date> element. The same tagged object format is returned by plistdecode for <date> elements, enabling seamless round-trips.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "rfc3339",
				Description: "An RFC 3339 timestamp string, e.g. \"2025-06-01T00:00:00Z\".",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *PlistDateFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))
	if resp.Error != nil {
		return
	}

	// Validate the timestamp is valid RFC 3339.
	_, err := time.Parse(time.RFC3339, input)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(
			"Invalid RFC 3339 timestamp: "+err.Error()))
		return
	}

	obj, err := makePlistTaggedObject(plistTypeDate, input)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(obj)))
}

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
		MarkdownDescription: "Returns a tagged object representing an NSData plist value. When passed to plistencode, this produces a <data> element. The same tagged object format is returned by plistdecode for <data> elements, enabling seamless round-trips. Use with filebase64() to embed binary data such as certificates.",
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

var _ function.Function = (*PlistRealFunction)(nil)

type PlistRealFunction struct{}

func NewPlistRealFunction() function.Function {
	return &PlistRealFunction{}
}

func (f *PlistRealFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "plistreal"
}

func (f *PlistRealFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Create a plist real (floating-point) value",
		MarkdownDescription: "Returns a tagged object representing a plist <real> value. Use this when you need to force a whole number to encode as <real> instead of <integer> in a plist. Fractional numbers like 3.14 are automatically encoded as <real> without needing this helper, but whole numbers like 2 would otherwise become <integer>. The same tagged object format is returned by plistdecode for whole-number <real> elements, enabling seamless round-trips.",
		Parameters: []function.Parameter{
			function.NumberParameter{
				Name:        "value",
				Description: "The numeric value for the <real> element.",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *PlistRealFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value *big.Float

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value))
	if resp.Error != nil {
		return
	}

	if value == nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Value must not be null."))
		return
	}

	f64, _ := value.Float64() // accuracy flag, not error
	str := strconv.FormatFloat(f64, 'f', -1, 64)

	obj, err := makePlistTaggedObject(plistTypeReal, str)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(obj)))
}
