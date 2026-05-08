package transform

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

/*
The transform package only operates on JSON-shaped values: null, bool, string, number, list, object. No dates, no binary blobs, no plist-tagged objects — JMESPath, JSONPath, JSON Patch, and JSON Merge Patch are all defined against the JSON data model.

The converter here is intentionally separate from internal/provider/dataformat/convert.go rather than shared. dataformat's converter has accumulated plist-aware behavior (it tags []byte and time.Time as `{"__plist_type": ...}` objects so plist round-trips preserve type), which is exactly what we DON'T want for query/patch operations defined against the JSON data model. Sharing would either pull plist semantics into transform or require option-flagging the shared code in two directions; the duplication is small and bounded, and behavior on overlapping inputs (the JSON value space) is verified by parallel unit tests in convert_test.go and dataformat/convert_test.go.
*/

const (
	// transformMaxDepth caps recursion in terraformToJSON / jsonToTerraform. JMESPath and JSONPath query engines have no internal depth bound; without this an adversarial input nested 10k+ levels would stack-OOM the goroutine. 1024 is generous — real configs rarely exceed 30.
	transformMaxDepth = 1024
	// transformMaxNodes caps the total node count traversed by terraformToJSON in a single call. JMESPath/JSONPath wildcards plus a million-element array can spend minutes searching at plan time. 1,000,000 is far above any realistic config; below that, query cost is bounded by the engine's internal complexity.
	transformMaxNodes = 1_000_000
)

// terraformToJSON converts a Terraform attr.Value to a Go interface{} drawn from the JSON value space (nil, bool, string, json.Number, []interface{}, map[string]interface{}). Numbers are returned as json.Number to preserve precision.
func terraformToJSON(v attr.Value) (interface{}, error) {
	nodes := 0
	return terraformToJSONImpl(v, 0, &nodes)
}

func terraformToJSONImpl(v attr.Value, depth int, nodes *int) (interface{}, error) {
	if depth >= transformMaxDepth {
		return nil, fmt.Errorf("input exceeds maximum supported nesting depth of %d", transformMaxDepth)
	}
	*nodes++
	if *nodes > transformMaxNodes {
		return nil, fmt.Errorf("input exceeds maximum supported node count of %d", transformMaxNodes)
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
		return bigFloatToJSONNumber(f), nil

	case basetypes.TupleValue:
		elements := val.Elements()
		out := make([]interface{}, len(elements))
		for i, elem := range elements {
			conv, err := terraformToJSONImpl(elem, depth+1, nodes)
			if err != nil {
				return nil, fmt.Errorf("index %d: %w", i, err)
			}
			out[i] = conv
		}
		return out, nil

	case basetypes.ListValue:
		elements := val.Elements()
		out := make([]interface{}, len(elements))
		for i, elem := range elements {
			conv, err := terraformToJSONImpl(elem, depth+1, nodes)
			if err != nil {
				return nil, fmt.Errorf("index %d: %w", i, err)
			}
			out[i] = conv
		}
		return out, nil

	case basetypes.SetValue:
		elements := val.Elements()
		out := make([]interface{}, len(elements))
		for i, elem := range elements {
			conv, err := terraformToJSONImpl(elem, depth+1, nodes)
			if err != nil {
				return nil, fmt.Errorf("index %d: %w", i, err)
			}
			out[i] = conv
		}
		return out, nil

	case basetypes.ObjectValue:
		attrs := val.Attributes()
		out := make(map[string]interface{}, len(attrs))
		for k, av := range attrs {
			conv, err := terraformToJSONImpl(av, depth+1, nodes)
			if err != nil {
				return nil, fmt.Errorf("key %q: %w", k, err)
			}
			out[k] = conv
		}
		return out, nil

	case basetypes.MapValue:
		elems := val.Elements()
		out := make(map[string]interface{}, len(elems))
		for k, av := range elems {
			conv, err := terraformToJSONImpl(av, depth+1, nodes)
			if err != nil {
				return nil, fmt.Errorf("key %q: %w", k, err)
			}
			out[k] = conv
		}
		return out, nil

	case basetypes.DynamicValue:
		return terraformToJSONImpl(val.UnderlyingValue(), depth, nodes)

	default:
		return nil, fmt.Errorf("unsupported Terraform type %T", v)
	}
}

// jsonToTerraform converts a Go value drawn from the JSON value space back to a Terraform attr.Value. It accepts the canonical encoding/json output (json.Number, []interface{}, map[string]interface{}) plus the standard numeric Go types so callers don't have to pre-normalize.
func jsonToTerraform(v interface{}) (attr.Value, error) {
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

	case float32:
		return types.NumberValue(big.NewFloat(float64(val))), nil

	case float64:
		if math.IsInf(val, 0) || math.IsNaN(val) {
			return nil, fmt.Errorf("non-finite number %v cannot be represented", val)
		}
		return types.NumberValue(big.NewFloat(val)), nil

	case int:
		return types.NumberValue(big.NewFloat(float64(val))), nil
	case int8:
		return types.NumberValue(big.NewFloat(float64(val))), nil
	case int16:
		return types.NumberValue(big.NewFloat(float64(val))), nil
	case int32:
		return types.NumberValue(big.NewFloat(float64(val))), nil
	case int64:
		return types.NumberValue(big.NewFloat(float64(val))), nil
	case uint:
		return types.NumberValue(new(big.Float).SetUint64(uint64(val))), nil
	case uint8:
		return types.NumberValue(big.NewFloat(float64(val))), nil
	case uint16:
		return types.NumberValue(big.NewFloat(float64(val))), nil
	case uint32:
		return types.NumberValue(big.NewFloat(float64(val))), nil
	case uint64:
		return types.NumberValue(new(big.Float).SetUint64(val)), nil

	case []interface{}:
		return jsonSliceToTuple(val)

	case map[string]interface{}:
		return jsonMapToObject(val)

	default:
		return nil, fmt.Errorf("unsupported Go type %T", v)
	}
}

func jsonSliceToTuple(slice []interface{}) (attr.Value, error) {
	if len(slice) == 0 {
		return types.TupleValueMust([]attr.Type{}, []attr.Value{}), nil
	}
	elemTypes := make([]attr.Type, len(slice))
	elemValues := make([]attr.Value, len(slice))
	for i, item := range slice {
		v, err := jsonToTerraform(item)
		if err != nil {
			return nil, fmt.Errorf("index %d: %w", i, err)
		}
		elemTypes[i] = v.Type(nil)
		elemValues[i] = v
	}
	return types.TupleValueMust(elemTypes, elemValues), nil
}

func jsonMapToObject(m map[string]interface{}) (attr.Value, error) {
	if len(m) == 0 {
		return types.ObjectValueMust(map[string]attr.Type{}, map[string]attr.Value{}), nil
	}
	attrTypes := make(map[string]attr.Type, len(m))
	attrValues := make(map[string]attr.Value, len(m))
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v, err := jsonToTerraform(m[k])
		if err != nil {
			return nil, fmt.Errorf("key %q: %w", k, err)
		}
		attrTypes[k] = v.Type(nil)
		attrValues[k] = v
	}
	obj, diags := types.ObjectValue(attrTypes, attrValues)
	if diags.HasError() {
		return nil, fmt.Errorf("creating object: %s", diags.Errors()[0].Detail())
	}
	return obj, nil
}

// bigFloatToJSONNumber renders a *big.Float as a json.Number. Integers print without a decimal point. Non-integers use 'f' formatting (no scientific notation) so the number renders identically to how a Terraform user would have typed it; 'g' would silently switch to "1.5e-08" for small magnitudes, surprising downstream consumers.
func bigFloatToJSONNumber(f *big.Float) json.Number {
	if f.IsInt() {
		i, _ := f.Int(nil)
		return json.Number(i.String())
	}
	return json.Number(f.Text('f', -1))
}
