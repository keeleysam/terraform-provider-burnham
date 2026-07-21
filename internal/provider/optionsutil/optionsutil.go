/*
Helpers shared by every provider function that takes a single optional options object via the framework's `VariadicParameter: function.DynamicParameter` pattern.

Conventions baked into the helpers:

  - Optional argument lives at parameter index 1 (one positional argument followed by `...options`). Errors returned here always tag index 1.
  - Exactly zero or one options object is allowed. More is an error.
  - When the caller passes `null` or an unknown value where an object is expected, the helper rejects it.
  - Per-key validation (allowed keys, value coercion) stays at the call site, because the schema differs per function.

These helpers do not migrate the older `dataformat/` family, which has a different "ignore unknown keys silently" contract; mixing them would change semantics. They cover the newer `identifiers/`, `text/`, and any future package adopting the same explicit schema.
*/

package optionsutil

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// attrToGoMaxDepth bounds recursion in AttrToGo so a pathologically nested options value cannot overflow the stack. Far deeper than any realistic hand-written HCL data tree.
const attrToGoMaxDepth = 200

// AttrToGo converts a Terraform attr.Value into a Go value in the JSON value space (nil, bool, string, json.Number, []any, map[string]any), suitable for json.Marshal. It is used to pass structured HCL data through to engines that consume JSON. Numbers become json.Number so integer-ness and precision survive the round trip. An unknown value (at any depth) is an error: an engine input must be a concrete plan value.
func AttrToGo(v attr.Value) (any, error) {
	return attrToGo(v, 0)
}

func attrToGo(v attr.Value, depth int) (any, error) {
	if depth > attrToGoMaxDepth {
		return nil, fmt.Errorf("value nested too deeply")
	}
	if v == nil || v.IsNull() {
		return nil, nil
	}
	if v.IsUnknown() {
		return nil, fmt.Errorf("value is unknown")
	}
	switch val := v.(type) {
	case basetypes.DynamicValue:
		return attrToGo(val.UnderlyingValue(), depth)
	case basetypes.BoolValue:
		return val.ValueBool(), nil
	case basetypes.StringValue:
		return val.ValueString(), nil
	case basetypes.NumberValue:
		return json.Number(val.ValueBigFloat().Text('f', -1)), nil
	case basetypes.ListValue:
		return elementsToGo(val.Elements(), depth)
	case basetypes.TupleValue:
		return elementsToGo(val.Elements(), depth)
	case basetypes.SetValue:
		return elementsToGo(val.Elements(), depth)
	case basetypes.ObjectValue:
		return attrsToGo(val.Attributes(), depth)
	case basetypes.MapValue:
		return attrsToGo(val.Elements(), depth)
	default:
		return nil, fmt.Errorf("unsupported value type %T", v)
	}
}

func elementsToGo(elems []attr.Value, depth int) ([]any, error) {
	out := make([]any, len(elems))
	for i, e := range elems {
		g, err := attrToGo(e, depth+1)
		if err != nil {
			return nil, err
		}
		out[i] = g
	}
	return out, nil
}

func attrsToGo(attrs map[string]attr.Value, depth int) (map[string]any, error) {
	out := make(map[string]any, len(attrs))
	for k, e := range attrs {
		g, err := attrToGo(e, depth+1)
		if err != nil {
			return nil, err
		}
		out[k] = g
	}
	return out, nil
}

// Base64List decodes a Terraform list/tuple/set of base64 strings into raw byte slices.
func Base64List(v attr.Value) ([][]byte, error) {
	elems, err := listElements(v)
	if err != nil {
		return nil, err
	}
	out := make([][]byte, 0, len(elems))
	for i, e := range elems {
		s, ok := e.(basetypes.StringValue)
		if !ok || s.IsNull() {
			return nil, fmt.Errorf("element %d is not a string", i)
		}
		b, err := base64.StdEncoding.DecodeString(s.ValueString())
		if err != nil {
			return nil, fmt.Errorf("element %d is not valid base64", i)
		}
		out = append(out, b)
	}
	return out, nil
}

// Base64StringMap extracts a Terraform map/object of base64 strings, keeping the values as base64 (the consumer decodes them) but validating that each is well-formed base64 so a bad value is reported here.
func Base64StringMap(v attr.Value) (map[string]string, error) {
	var attrs map[string]attr.Value
	switch val := v.(type) {
	case basetypes.MapValue:
		attrs = val.Elements()
	case basetypes.ObjectValue:
		attrs = val.Attributes()
	default:
		return nil, fmt.Errorf("not a map or object")
	}
	out := make(map[string]string, len(attrs))
	for k, e := range attrs {
		s, ok := e.(basetypes.StringValue)
		if !ok || s.IsNull() {
			return nil, fmt.Errorf("value for %q is not a string", k)
		}
		if _, err := base64.StdEncoding.DecodeString(s.ValueString()); err != nil {
			return nil, fmt.Errorf("value for %q is not valid base64", k)
		}
		out[k] = s.ValueString()
	}
	return out, nil
}

func listElements(v attr.Value) ([]attr.Value, error) {
	switch val := v.(type) {
	case basetypes.ListValue:
		return val.Elements(), nil
	case basetypes.TupleValue:
		return val.Elements(), nil
	case basetypes.SetValue:
		return val.Elements(), nil
	default:
		return nil, fmt.Errorf("not a list")
	}
}

// SingleOptionsObject pulls the (zero-or-one) options object out of a `VariadicParameter` slice and returns its attribute map. Returns `(nil, nil)` when the caller passed no options, and the caller should fall through to its defaults. `hint` is a snippet shown in the error when the caller passes something that isn't an object literal: typically a worked example like `"{ size = 10 }"`.
func SingleOptionsObject(opts []types.Dynamic, hint string) (map[string]attr.Value, *function.FuncError) {
	if len(opts) == 0 {
		return nil, nil
	}
	if len(opts) > 1 {
		return nil, function.NewArgumentFuncError(1, "at most one options argument may be provided")
	}
	obj, ok := opts[0].UnderlyingValue().(basetypes.ObjectValue)
	if !ok || obj.IsNull() || obj.IsUnknown() {
		return nil, function.NewArgumentFuncError(1, fmt.Sprintf("options must be an object literal, e.g. %s", hint))
	}
	return obj.Attributes(), nil
}

// NumberAttrToInt converts a Terraform Number attribute (carries a `*big.Float` internally) into a Go int. Errors when the value is null/unknown, non-integral, or out of int range. Lossy conversions never happen: Terraform numbers preserve the integer-ness of their input.
func NumberAttrToInt(v attr.Value) (int, error) {
	num, ok := v.(basetypes.NumberValue)
	if !ok {
		return 0, fmt.Errorf("expected a number, got %T", v)
	}
	if num.IsNull() || num.IsUnknown() {
		return 0, fmt.Errorf("value is null or unknown")
	}
	bi, accuracy := num.ValueBigFloat().Int(nil)
	if accuracy != big.Exact {
		return 0, fmt.Errorf("not a whole number")
	}
	if !bi.IsInt64() {
		return 0, fmt.Errorf("out of int range")
	}
	return int(bi.Int64()), nil
}
