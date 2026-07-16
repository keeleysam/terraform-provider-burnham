package dataformat

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vmihailenco/msgpack/v5"
	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

// msgpackMaxCollectionElements caps how many array elements or map pairs we will
// allocate for from a single length header, mirroring the CBOR decoder's
// MaxArrayElements / MaxMapPairs (131072). MessagePack length prefixes are up to
// 32 bits wide, so an 8-byte input can claim 2^32-1 elements and drive tens of GB
// of allocation. The msgpack/v5 library does not cap the []interface{} slice path
// at all, and our own map decoder sizes the map directly from the header, so we
// enforce the bound ourselves and return a decode error rather than allocating
// unboundedly.
const msgpackMaxCollectionElements = 131072

var _ function.Function = (*MsgpackDecodeFunction)(nil)

type MsgpackDecodeFunction struct{}

func NewMsgpackDecodeFunction() function.Function { return &MsgpackDecodeFunction{} }

func (f *MsgpackDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "msgpackdecode"
}

func (f *MsgpackDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Decode a base64-encoded MessagePack blob into a value",
		MarkdownDescription: "Decodes [MessagePack](https://msgpack.org/) bytes (provided as a standard base64 string, since HCL strings are UTF-8 only) into a Terraform value.\n\nMessagePack maps become objects, arrays become tuples, integers and floats become numbers. Binary blobs (msgpack `bin` format) are decoded to base64 strings. Extension types are not supported.\n\n**Common uses:** consuming msgpack-encoded payloads from caches (Redis, etcd), inspecting `kubectl get --raw` output, or round-tripping fixtures.",
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
	goVal, err := decodeMsgpackValue(dec)
	if err != nil {
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

// decodeMsgpackValue decodes one value, bounding array allocations by the
// element cap before delegating scalars and maps to the library. The msgpack/v5
// []interface{} path (DecodeInterface -> decodeSlice) sizes its backing array
// straight from the header with no limit, so we intercept arrays here and read
// them element by element after checking the count.
func decodeMsgpackValue(d *msgpack.Decoder) (interface{}, error) {
	c, err := d.PeekCode()
	if err != nil {
		return nil, err
	}
	if msgpcode.IsFixedArray(c) || c == msgpcode.Array16 || c == msgpcode.Array32 {
		n, err := d.DecodeArrayLen()
		if err != nil {
			return nil, err
		}
		if n < 0 {
			return nil, nil
		}
		if n > msgpackMaxCollectionElements {
			return nil, fmt.Errorf("array length %d exceeds maximum of %d elements", n, msgpackMaxCollectionElements)
		}
		out := make([]interface{}, 0, n)
		for i := 0; i < n; i++ {
			v, err := decodeMsgpackValue(d)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	}
	// Maps route through decodeMsgpackMapStringInterface via the map decoder hook; everything else (scalars, bin, ext) uses the library's own logic.
	return d.DecodeInterface()
}

// decodeMsgpackMapStringInterface forces map keys to strings; without this, msgpack/v5 returns map[interface{}]interface{} which goToTerraformValue can't handle.
func decodeMsgpackMapStringInterface(d *msgpack.Decoder) (interface{}, error) {
	n, err := d.DecodeMapLen()
	if err != nil {
		return nil, err
	}
	if n < 0 {
		return nil, nil
	}
	if n > msgpackMaxCollectionElements {
		return nil, fmt.Errorf("map length %d exceeds maximum of %d pairs", n, msgpackMaxCollectionElements)
	}
	out := make(map[string]interface{}, n)
	for i := 0; i < n; i++ {
		k, err := d.DecodeString()
		if err != nil {
			return nil, err
		}
		v, err := decodeMsgpackValue(d)
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
		Summary:             "Encode a value as base64 MessagePack",
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
	if unknownStringResultIfNeeded(ctx, resp, value.UnderlyingValue(), nil) {
		return
	}

	goVal, err := terraformValueToGo(value.UnderlyingValue(), false)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to convert value: "+err.Error())
		return
	}

	prepared := goValueForBinaryEncode(goVal)

	// UseCompactInts emits the smallest fixint/int* form that fits the value, matching MessagePack's space-efficiency intent. Without it, every integer becomes an 8-byte int64 regardless of magnitude, which is wasteful and produces different bytes for the same numeric value depending on Go's static type. SetSortMapKeys produces stable byte output regardless of input map iteration order, so users can compare or hash encoded payloads safely.
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
