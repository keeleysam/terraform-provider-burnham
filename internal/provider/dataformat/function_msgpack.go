package dataformat

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vmihailenco/msgpack/v5"
)

var _ function.Function = (*MsgpackDecodeFunction)(nil)

type MsgpackDecodeFunction struct{}

func NewMsgpackDecodeFunction() function.Function { return &MsgpackDecodeFunction{} }

func (f *MsgpackDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "msgpackdecode"
}

func (f *MsgpackDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Decode a base64-encoded MessagePack blob into a value",
		MarkdownDescription: "Decodes [MessagePack](https://msgpack.org/) bytes — provided as a standard base64 string, since HCL strings are UTF-8 only — into a Terraform value.\n\nMessagePack maps become objects, arrays become tuples, integers and floats become numbers. Binary blobs (msgpack `bin` format) are decoded to base64 strings. Extension types are not supported.\n\n**Common uses:** consuming msgpack-encoded payloads from caches (Redis, etcd), inspecting `kubectl get --raw` output, or round-tripping fixtures.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "input",
				Description: "A base64-encoded MessagePack blob.",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *MsgpackDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
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

	dec := msgpack.NewDecoder(bytes.NewReader(raw))
	dec.SetMapDecoder(decodeMsgpackMapStringInterface)
	var goVal interface{}
	if err := dec.Decode(&goVal); err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to decode MessagePack: "+err.Error())
		return
	}

	tfVal, err := goToTerraformValue(goVal)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to convert value: "+err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(tfVal)))
}

// decodeMsgpackMapStringInterface forces map keys to strings — without this, msgpack/v5 returns map[interface{}]interface{} which goToTerraformValue can't handle.
func decodeMsgpackMapStringInterface(d *msgpack.Decoder) (interface{}, error) {
	n, err := d.DecodeMapLen()
	if err != nil {
		return nil, err
	}
	if n < 0 {
		return nil, nil
	}
	out := make(map[string]interface{}, n)
	for i := 0; i < n; i++ {
		k, err := d.DecodeString()
		if err != nil {
			return nil, err
		}
		v, err := d.DecodeInterface()
		if err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, nil
}

var _ function.Function = (*MsgpackEncodeFunction)(nil)

type MsgpackEncodeFunction struct{}

func NewMsgpackEncodeFunction() function.Function { return &MsgpackEncodeFunction{} }

func (f *MsgpackEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "msgpackencode"
}

func (f *MsgpackEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Encode a value as base64 MessagePack",
		MarkdownDescription: "Encodes a Terraform value as [MessagePack](https://msgpack.org/) and returns the result as a standard base64 string. Object keys are written in sorted order for stable output. Whole-number floats are emitted as integers (matching the conventions of `jsonencode` here).\n\n**Common uses:** generating msgpack payloads to seed Redis fixtures, write to disk via `local_file`, or feed external tooling.",
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "value",
				Description: "The value to encode.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *MsgpackEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value))
	if resp.Error != nil {
		return
	}

	goVal, err := terraformValueToGo(value.UnderlyingValue(), false)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to convert value: "+err.Error())
		return
	}

	prepared := goValueForBinaryEncode(goVal)

	// UseCompactInts emits the smallest fixint/int* form that fits the value, matching MessagePack's space-efficiency intent — without it, every integer becomes an 8-byte int64 regardless of magnitude, which is wasteful and produces different bytes for the same numeric value depending on Go's static type. SetSortMapKeys produces stable byte output regardless of input map iteration order, so users can compare or hash encoded payloads safely.
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	enc.UseCompactInts(true)
	enc.SetSortMapKeys(true)
	if err := enc.Encode(prepared); err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to encode MessagePack: "+err.Error()))
		return
	}
	out := buf.Bytes()

	encoded := base64.StdEncoding.EncodeToString(out)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, encoded))
}
