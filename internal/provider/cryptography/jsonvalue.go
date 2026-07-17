/*
JSON value bridging for the JOSE family (jwt_*, jwk_*).

The JWT and JWK functions take and return dynamic objects (claims, headers, JWK members) whose shape is not known at schema-definition time. These helpers convert between the Terraform value space and the Go/JSON value space:

  - terraformToGoJSON walks a Terraform attr.Value into a Go interface{} tree using the JSON value space (nil, bool, string, json.Number, []interface{}, map[string]interface{}). Numbers are carried as json.Number so json.Marshal emits them as bare number tokens with full precision, never as quoted strings (which is what *big.Int would otherwise produce through encoding.TextMarshaler).
  - goJSONToTerraform walks a json.Unmarshal (UseNumber) result back into a Terraform attr.Value.
  - hasUnknownValue mirrors the dataformat package's nested-unknown detection so the JOSE functions honour the framework's unknown-value contract.

These deliberately duplicate the shape of the dataformat package's converters rather than importing them: the two families evolve independently and the JSON-only value space here is a strict subset (no plist tagging, no time.Time / []byte special-casing) that would be awkward to express through the dataformat entry points.
*/

package cryptography

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// joseConvertMaxDepth caps recursion in the converters so an adversarial deeply-nested claims object or JWK cannot stack-OOM the provider at plan time. 512 is far above any realistic JWT payload or JWK set.
const joseConvertMaxDepth = 512

// hasUnknownValue reports whether v holds an unknown value at any depth. Terraform only auto-defers a function call when a whole argument is unknown, so a known container with an unknown nested value still reaches Run; callers short-circuit to an unknown result rather than baking a concrete value that changes at apply.
func hasUnknownValue(v attr.Value) bool {
	if v == nil {
		return false
	}
	if v.IsUnknown() {
		return true
	}
	switch val := v.(type) {
	case basetypes.DynamicValue:
		return hasUnknownValue(val.UnderlyingValue())
	case basetypes.TupleValue:
		return sliceHasUnknown(val.Elements())
	case basetypes.ListValue:
		return sliceHasUnknown(val.Elements())
	case basetypes.SetValue:
		return sliceHasUnknown(val.Elements())
	case basetypes.ObjectValue:
		return mapHasUnknown(val.Attributes())
	case basetypes.MapValue:
		return mapHasUnknown(val.Elements())
	}
	return false
}

func sliceHasUnknown(elems []attr.Value) bool {
	for _, e := range elems {
		if hasUnknownValue(e) {
			return true
		}
	}
	return false
}

func mapHasUnknown(attrs map[string]attr.Value) bool {
	for _, a := range attrs {
		if hasUnknownValue(a) {
			return true
		}
	}
	return false
}

// terraformToGoJSON converts a Terraform attr.Value to a Go interface{} in the JSON value space.
func terraformToGoJSON(v attr.Value) (interface{}, error) {
	return terraformToGoJSONImpl(v, 0)
}

func terraformToGoJSONImpl(v attr.Value, depth int) (interface{}, error) {
	if depth >= joseConvertMaxDepth {
		return nil, fmt.Errorf("value exceeds maximum supported nesting depth of %d", joseConvertMaxDepth)
	}
	if v == nil || v.IsNull() || v.IsUnknown() {
		return nil, nil
	}
	switch val := v.(type) {
	case basetypes.BoolValue:
		return val.ValueBool(), nil
	case basetypes.StringValue:
		return val.ValueString(), nil
	case basetypes.NumberValue:
		f := val.ValueBigFloat()
		if f == nil {
			return json.Number("0"), nil
		}
		if f.IsInt() {
			i, _ := f.Int(nil)
			return json.Number(i.String()), nil
		}
		return json.Number(f.Text('g', -1)), nil
	case basetypes.TupleValue:
		return sliceToGoJSON(val.Elements(), depth)
	case basetypes.ListValue:
		return sliceToGoJSON(val.Elements(), depth)
	case basetypes.SetValue:
		return sliceToGoJSON(val.Elements(), depth)
	case basetypes.ObjectValue:
		return mapToGoJSON(val.Attributes(), depth)
	case basetypes.MapValue:
		return mapToGoJSON(val.Elements(), depth)
	case basetypes.DynamicValue:
		return terraformToGoJSONImpl(val.UnderlyingValue(), depth)
	default:
		return nil, fmt.Errorf("unsupported Terraform type %T", v)
	}
}

func sliceToGoJSON(elems []attr.Value, depth int) ([]interface{}, error) {
	out := make([]interface{}, len(elems))
	for i, e := range elems {
		gv, err := terraformToGoJSONImpl(e, depth+1)
		if err != nil {
			return nil, fmt.Errorf("index %d: %w", i, err)
		}
		out[i] = gv
	}
	return out, nil
}

func mapToGoJSON(attrs map[string]attr.Value, depth int) (map[string]interface{}, error) {
	out := make(map[string]interface{}, len(attrs))
	for k, a := range attrs {
		gv, err := terraformToGoJSONImpl(a, depth+1)
		if err != nil {
			return nil, fmt.Errorf("key %q: %w", k, err)
		}
		out[k] = gv
	}
	return out, nil
}

// goJSONToTerraform converts a json.Unmarshal result (decoded with UseNumber) into a Terraform attr.Value in the JSON value space.
func goJSONToTerraform(v interface{}) (attr.Value, error) {
	return goJSONToTerraformImpl(v, 0)
}

func goJSONToTerraformImpl(v interface{}, depth int) (attr.Value, error) {
	if depth >= joseConvertMaxDepth {
		return nil, fmt.Errorf("value exceeds maximum supported nesting depth of %d", joseConvertMaxDepth)
	}
	switch val := v.(type) {
	case nil:
		return types.DynamicNull(), nil
	case bool:
		return types.BoolValue(val), nil
	case string:
		return types.StringValue(val), nil
	case json.Number:
		f, _, err := big.NewFloat(0).Parse(string(val), 10)
		if err != nil {
			return nil, fmt.Errorf("invalid json.Number %q: %w", val, err)
		}
		return types.NumberValue(f), nil
	case float64:
		return types.NumberValue(big.NewFloat(val)), nil
	case []interface{}:
		return goSliceToTerraform(val, depth)
	case map[string]interface{}:
		return goMapToTerraform(val, depth)
	default:
		return nil, fmt.Errorf("unsupported JSON value type %T", v)
	}
}

func goSliceToTerraform(slice []interface{}, depth int) (attr.Value, error) {
	if len(slice) == 0 {
		return types.TupleValueMust([]attr.Type{}, []attr.Value{}), nil
	}
	elemTypes := make([]attr.Type, len(slice))
	elemVals := make([]attr.Value, len(slice))
	for i, item := range slice {
		ev, err := goJSONToTerraformImpl(item, depth+1)
		if err != nil {
			return nil, fmt.Errorf("index %d: %w", i, err)
		}
		elemTypes[i] = ev.Type(nil)
		elemVals[i] = ev
	}
	return types.TupleValueMust(elemTypes, elemVals), nil
}

func goMapToTerraform(m map[string]interface{}, depth int) (attr.Value, error) {
	if len(m) == 0 {
		return types.ObjectValueMust(map[string]attr.Type{}, map[string]attr.Value{}), nil
	}
	attrTypes := make(map[string]attr.Type, len(m))
	attrVals := make(map[string]attr.Value, len(m))
	for k, item := range m {
		ev, err := goJSONToTerraformImpl(item, depth+1)
		if err != nil {
			return nil, fmt.Errorf("key %q: %w", k, err)
		}
		attrTypes[k] = ev.Type(nil)
		attrVals[k] = ev
	}
	obj, diags := types.ObjectValue(attrTypes, attrVals)
	if diags.HasError() {
		return nil, fmt.Errorf("creating object: %s", diagsToString(diags))
	}
	return obj, nil
}

// jsonBytesToTerraform decodes JSON bytes (with UseNumber) into a Terraform attr.Value. Shared by jwt_decode / jwt_verify for header and payload, and by jwk_encode for the JWK object.
func jsonBytesToTerraform(b []byte) (attr.Value, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var raw interface{}
	if err := dec.Decode(&raw); err != nil {
		return nil, err
	}
	return goJSONToTerraform(raw)
}
