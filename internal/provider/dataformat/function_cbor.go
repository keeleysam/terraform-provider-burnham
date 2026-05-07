package dataformat

import (
	"context"
	"encoding/base64"
	"reflect"

	"github.com/fxamacker/cbor/v2"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// cborMapStringInterfaceType pins decoded CBOR maps to map[string]interface{} so goToTerraformValue's existing path handles them directly. CBOR allows non-string keys; if encountered, the cbor library will return an error during unmarshal.
var cborMapStringInterfaceType = reflect.TypeOf(map[string]interface{}{})

var _ function.Function = (*CBORDecodeFunction)(nil)

type CBORDecodeFunction struct{}

func NewCBORDecodeFunction() function.Function { return &CBORDecodeFunction{} }

func (f *CBORDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cbordecode"
}

func (f *CBORDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Decode a base64-encoded CBOR blob into a value",
		MarkdownDescription: "Decodes [CBOR](https://www.rfc-editor.org/rfc/rfc8949) ([RFC 8949](https://www.rfc-editor.org/rfc/rfc8949)) bytes — provided as a standard base64 string, since HCL strings are UTF-8 only — into a Terraform value.\n\n" +
			"Type mapping: CBOR maps with string keys become objects (maps with non-string keys are an error); arrays become tuples; integers and floats become numbers; byte strings become standard base64 strings; tag-0/tag-1 datetimes become [RFC 3339](https://www.rfc-editor.org/rfc/rfc3339) strings; bignum tags (2/3) become full-precision numbers (Terraform's number type uses arbitrary-precision big floats).\n\n" +
			"Backed by [fxamacker/cbor](https://github.com/fxamacker/cbor), an RFC 8949 conforming implementation.\n\n" +
			"**Common uses:** consuming CBOR-encoded payloads from IoT/CoAP gateways, COSE signed objects, or any binary structured-data feed where compactness matters more than human readability.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "input",
				Description: "A base64-encoded CBOR blob.",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *CBORDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))
	if resp.Error != nil {
		return
	}

	raw, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Invalid base64: "+err.Error()))
		return
	}

	// DefaultMapType: map[string]interface{} so goToTerraformValue's existing case handles maps directly.
	// TimeTag: DecTagOptional makes tag-0 (RFC 3339) and tag-1 (epoch) datetimes decode to time.Time, which goValueDecodeBinary then converts to an RFC 3339 string. Without this, tag-1 epoch tags would silently come through as a bare number.
	decMode, err := cbor.DecOptions{
		DefaultMapType: cborMapStringInterfaceType,
		TimeTag:        cbor.DecTagOptional,
	}.DecMode()
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("CBOR decoder setup failed: "+err.Error()))
		return
	}

	var goVal interface{}
	if err := decMode.Unmarshal(raw, &goVal); err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to decode CBOR: "+err.Error()))
		return
	}

	tfVal, err := goToTerraformValue(goVal)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to convert value: "+err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(tfVal)))
}

var _ function.Function = (*CBOREncodeFunction)(nil)

type CBOREncodeFunction struct{}

func NewCBOREncodeFunction() function.Function { return &CBOREncodeFunction{} }

func (f *CBOREncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cborencode"
}

func (f *CBOREncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Encode a value as base64 CBOR",
		MarkdownDescription: "Encodes a Terraform value as [CBOR](https://www.rfc-editor.org/rfc/rfc8949) ([RFC 8949](https://www.rfc-editor.org/rfc/rfc8949)) and returns the result as a standard base64 string. " +
			"Output uses CBOR's [Core Deterministic Encoding](https://www.rfc-editor.org/rfc/rfc8949#section-4.2.1): definite-length items, sorted map keys, and shortest-form integers — so the same input produces byte-identical output.\n\n" +
			"Whole-number floats are emitted as integers (matching the conventions of `jsonencode` here). Strings are encoded as CBOR text strings; the function does not synthesize byte strings or tagged values from HCL inputs.\n\n" +
			"Backed by [fxamacker/cbor](https://github.com/fxamacker/cbor).\n\n" +
			"**Common uses:** generating CBOR fixtures for IoT services, COSE-style payloads, or any binary feed that benefits from a deterministic encoding.",
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "value",
				Description: "The value to encode.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *CBOREncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value))
	if resp.Error != nil {
		return
	}

	goVal, err := terraformValueToGo(value.UnderlyingValue(), false)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to convert value: "+err.Error()))
		return
	}

	prepared := goValueForBinaryEncode(goVal)

	encMode, err := cbor.CoreDetEncOptions().EncMode()
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("CBOR encoder setup failed: "+err.Error()))
		return
	}

	out, err := encMode.Marshal(prepared)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to encode CBOR: "+err.Error()))
		return
	}

	encoded := base64.StdEncoding.EncodeToString(out)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, encoded))
}
