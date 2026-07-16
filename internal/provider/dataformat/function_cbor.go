package dataformat

import (
	_ "embed"

	"context"
	"encoding/base64"
	"fmt"
	"reflect"

	"github.com/fxamacker/cbor/v2"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// cborMapStringInterfaceType pins decoded CBOR maps to map[string]interface{} so goToTerraformValue's existing path handles them directly. CBOR allows non-string keys; if encountered, the cbor library will return an error during unmarshal.
var cborMapStringInterfaceType = reflect.TypeOf(map[string]interface{}{})

var _ function.Function = (*CBORDecodeFunction)(nil)

//go:embed descriptions/cbordecode.md
var cbordecodeDescription string

type CBORDecodeFunction struct{}

func NewCBORDecodeFunction() function.Function { return &CBORDecodeFunction{} }

func (f *CBORDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cbordecode"
}

func (f *CBORDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Decode a base64-encoded CBOR blob into a value",
		MarkdownDescription: cbordecodeDescription,
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

	if len(input) > dataformatMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("input exceeds maximum supported length of %d bytes", dataformatMaxInputBytes))
		return
	}
	raw, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "invalid base64: "+err.Error())
		return
	}

	// DefaultMapType: map[string]interface{} so goToTerraformValue's existing case handles maps directly.
	// TimeTag: DecTagOptional makes tag-0 (RFC 3339) and tag-1 (epoch) datetimes decode to time.Time, which goValueDecodeBinary then converts to an RFC 3339 string. Without this, tag-1 epoch tags would silently come through as a bare number.
	// MaxNestedLevels / MaxArrayElements / MaxMapPairs: defensive caps so an adversarial CBOR blob with millions of nested items or a million-element array can't OOM the Terraform process at plan time. RFC 8949 places no upper bound on these, but fxamacker/cbor exposes the knobs and applies a default of 32 / 128k / 128k; we lift `MaxNestedLevels` to 256 (well above any realistic config) and keep the element-count default since it already bounds memory.
	decMode, err := cbor.DecOptions{
		DefaultMapType:   cborMapStringInterfaceType,
		TimeTag:          cbor.DecTagOptional,
		MaxNestedLevels:  256,
		MaxArrayElements: 131072,
		MaxMapPairs:      131072,
	}.DecMode()
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("CBOR decoder setup failed: "+err.Error()))
		return
	}

	var goVal interface{}
	if err := decMode.Unmarshal(raw, &goVal); err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to decode CBOR: "+err.Error())
		return
	}

	tfVal, err := goToTerraformValue(goVal)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to convert value: "+err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(tfVal)))
}

var _ function.Function = (*CBOREncodeFunction)(nil)

//go:embed descriptions/cborencode.md
var cborencodeDescription string

type CBOREncodeFunction struct{}

func NewCBOREncodeFunction() function.Function { return &CBOREncodeFunction{} }

func (f *CBOREncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cborencode"
}

func (f *CBOREncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Encode a value as base64 CBOR",
		MarkdownDescription: cborencodeDescription,
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
	if unknownStringResultIfNeeded(ctx, resp, value.UnderlyingValue(), nil) {
		return
	}

	goVal, err := terraformValueToGo(value.UnderlyingValue(), false)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to convert value: "+err.Error())
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
