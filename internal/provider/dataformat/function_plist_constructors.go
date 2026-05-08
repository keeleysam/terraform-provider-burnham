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
		Summary:             "Create a plist date value from an RFC 3339 timestamp",
		MarkdownDescription: "Returns a tagged object representing an [`NSDate`](https://developer.apple.com/documentation/foundation/nsdate) plist value. When passed to `plistencode`, this produces a `<date>` XML element with the given timestamp. `plistdecode` returns the same tagged-object shape for `<date>` elements, so encode/decode round-trips preserve the type.\n\nThe input must be an [RFC 3339](https://www.rfc-editor.org/rfc/rfc3339) timestamp string (e.g. `\"2026-06-01T00:00:00Z\"`).\n\n**Common uses:** setting `RemovalDate`, `ExpirationDate`, or other date-typed fields in Apple configuration profiles.",
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
		Summary:             "Create a plist data value from a base64-encoded string",
		MarkdownDescription: "Returns a tagged object representing an [`NSData`](https://developer.apple.com/documentation/foundation/nsdata) plist value. When passed to `plistencode`, this produces a `<data>` XML element with the given binary payload. `plistdecode` returns the same tagged-object shape for `<data>` elements, so encode/decode round-trips preserve the type.\n\nThe input must be a base64-encoded string. Pair with `filebase64(\"path/to/file\")` to embed file contents like certificates, profile-signing material, or images.\n\n**Common uses:** embedding signing certificates, custom icons, or other binary blobs into Apple configuration profiles.",
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
		Summary:             "Create a plist real (floating-point) value",
		MarkdownDescription: "Returns a tagged object representing a plist `<real>` (floating-point) value. `plistencode` would normally encode whole numbers as `<integer>` and only fractional numbers as `<real>` — this helper forces a whole number into `<real>` form when the consumer expects a floating-point type. `plistdecode` returns the same tagged-object shape for whole-number `<real>` elements, preserving the type across round-trips.\n\nFractional values like `3.14` already encode as `<real>` automatically — this helper is only needed for whole-number reals.\n\n**Common uses:** profile fields that demand a floating-point type even when the value happens to be a whole number (e.g. some `Rating`, `Score`, or version fields in MDM payloads).",
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
