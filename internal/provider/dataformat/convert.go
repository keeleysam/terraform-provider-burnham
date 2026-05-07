package dataformat

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	plistTypeKey  = "__plist_type"
	plistValueKey = "value"
	plistTypeDate = "date"
	plistTypeData = "data"
	plistTypeReal = "real"
)

// goToTerraformValue converts a Go interface{} (as returned by json.Unmarshal or a binary-format decoder) to a Terraform attr.Value, using the JSON value space: null, bool, string, number, list, object. []byte is rendered as a base64 string, time.Time as an RFC 3339 string, big.Int / *big.Int as an arbitrary-precision number. Use goToTerraformValuePlist when round-tripping plist-tagged types.
func goToTerraformValue(v interface{}) (attr.Value, error) {
	return goToTerraformValueImpl(v, false)
}

// goToTerraformValuePlist is the plist-aware variant: time.Time, []byte, and whole-number float64 emit __plist_type-tagged objects so plistencode can later round-trip them back to <date>, <data>, and <real> elements respectively. Use goToTerraformValue everywhere else — for non-plist callers, the tagged objects would look like garbage map members in their HCL output.
func goToTerraformValuePlist(v interface{}) (attr.Value, error) {
	return goToTerraformValueImpl(v, true)
}

func goToTerraformValueImpl(v interface{}, plist bool) (attr.Value, error) {
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
		// In plist mode, whole-number floats need a tagged object to distinguish them from <integer> during round-trips. In default mode, they're just numbers.
		if plist && val == math.Trunc(val) && !math.IsInf(val, 0) && !math.IsNaN(val) {
			return makePlistTaggedObject(plistTypeReal, strconv.FormatFloat(val, 'f', -1, 64))
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

	case big.Int:
		return types.NumberValue(new(big.Float).SetInt(&val)), nil

	case *big.Int:
		return types.NumberValue(new(big.Float).SetInt(val)), nil

	case time.Time:
		if plist {
			return makePlistTaggedObject(plistTypeDate, val.UTC().Format(time.RFC3339))
		}
		return types.StringValue(val.UTC().Format(time.RFC3339)), nil

	case []byte:
		if plist {
			return makePlistTaggedObject(plistTypeData, base64.StdEncoding.EncodeToString(val))
		}
		return types.StringValue(base64.StdEncoding.EncodeToString(val)), nil

	case []interface{}:
		return goSliceToTupleImpl(val, plist)

	case map[string]interface{}:
		return goMapToObjectImpl(val, plist)

	default:
		return nil, fmt.Errorf("unsupported Go type %T", v)
	}
}

// makePlistTaggedObject creates a Terraform object with __plist_type and value keys.
func makePlistTaggedObject(plistType, value string) (attr.Value, error) {
	attrTypes := map[string]attr.Type{
		plistTypeKey:  types.StringType,
		plistValueKey: types.StringType,
	}
	attrValues := map[string]attr.Value{
		plistTypeKey:  types.StringValue(plistType),
		plistValueKey: types.StringValue(value),
	}
	obj, diags := types.ObjectValue(attrTypes, attrValues)
	if diags.HasError() {
		return nil, fmt.Errorf("creating tagged object: %s", diags.Errors()[0].Detail())
	}
	return obj, nil
}

func goSliceToTuple(slice []interface{}) (attr.Value, error) {
	return goSliceToTupleImpl(slice, false)
}

func goSliceToTupleImpl(slice []interface{}, plist bool) (attr.Value, error) {
	if len(slice) == 0 {
		return types.TupleValueMust([]attr.Type{}, []attr.Value{}), nil
	}

	elemTypes := make([]attr.Type, len(slice))
	elemValues := make([]attr.Value, len(slice))

	for i, item := range slice {
		val, err := goToTerraformValueImpl(item, plist)
		if err != nil {
			return nil, fmt.Errorf("index %d: %w", i, err)
		}
		elemTypes[i] = val.Type(nil)
		elemValues[i] = val
	}

	return types.TupleValueMust(elemTypes, elemValues), nil
}

func goMapToObject(m map[string]interface{}) (attr.Value, error) {
	return goMapToObjectImpl(m, false)
}

func goMapToObjectImpl(m map[string]interface{}, plist bool) (attr.Value, error) {
	if len(m) == 0 {
		return types.ObjectValueMust(map[string]attr.Type{}, map[string]attr.Value{}), nil
	}

	attrTypes := make(map[string]attr.Type, len(m))
	attrValues := make(map[string]attr.Value, len(m))

	for k, v := range m {
		val, err := goToTerraformValueImpl(v, plist)
		if err != nil {
			return nil, fmt.Errorf("key %q: %w", k, err)
		}
		attrTypes[k] = val.Type(nil)
		attrValues[k] = val
	}

	obj, diags := types.ObjectValue(attrTypes, attrValues)
	if diags.HasError() {
		return nil, fmt.Errorf("creating object: %s", diags.Errors()[0].Detail())
	}
	return obj, nil
}

// terraformValueToGo converts a Terraform attr.Value back to a Go interface{}.
// When plistMode is true, tagged objects with __plist_type are converted to
// their native Go types (time.Time, []byte).
func terraformValueToGo(v attr.Value, plistMode bool) (interface{}, error) {
	if v.IsNull() || v.IsUnknown() {
		return nil, nil
	}

	switch val := v.(type) {
	case basetypes.BoolValue:
		return val.ValueBool(), nil

	case basetypes.StringValue:
		return val.ValueString(), nil

	case basetypes.NumberValue:
		f := val.ValueBigFloat()
		// Return as float64 — the caller decides integer vs real based on value.
		result, _ := f.Float64()
		return result, nil

	case basetypes.TupleValue:
		elements := val.Elements()
		slice := make([]interface{}, len(elements))
		for i, elem := range elements {
			goVal, err := terraformValueToGo(elem, plistMode)
			if err != nil {
				return nil, fmt.Errorf("index %d: %w", i, err)
			}
			slice[i] = goVal
		}
		return slice, nil

	case basetypes.ObjectValue:
		attrs := val.Attributes()

		// Check for plist tagged objects
		if plistMode {
			if goVal, ok, err := tryUnpackPlistTaggedObject(attrs); ok || err != nil {
				return goVal, err
			}
		}

		m := make(map[string]interface{}, len(attrs))
		for k, attrVal := range attrs {
			goVal, err := terraformValueToGo(attrVal, plistMode)
			if err != nil {
				return nil, fmt.Errorf("key %q: %w", k, err)
			}
			m[k] = goVal
		}
		return m, nil

	case basetypes.ListValue:
		elements := val.Elements()
		slice := make([]interface{}, len(elements))
		for i, elem := range elements {
			goVal, err := terraformValueToGo(elem, plistMode)
			if err != nil {
				return nil, fmt.Errorf("index %d: %w", i, err)
			}
			slice[i] = goVal
		}
		return slice, nil

	case basetypes.MapValue:
		elems := val.Elements()
		m := make(map[string]interface{}, len(elems))
		for k, elem := range elems {
			goVal, err := terraformValueToGo(elem, plistMode)
			if err != nil {
				return nil, fmt.Errorf("key %q: %w", k, err)
			}
			m[k] = goVal
		}
		return m, nil

	case basetypes.SetValue:
		elements := val.Elements()
		slice := make([]interface{}, len(elements))
		for i, elem := range elements {
			goVal, err := terraformValueToGo(elem, plistMode)
			if err != nil {
				return nil, fmt.Errorf("index %d: %w", i, err)
			}
			slice[i] = goVal
		}
		return slice, nil

	case basetypes.DynamicValue:
		return terraformValueToGo(val.UnderlyingValue(), plistMode)

	default:
		return nil, fmt.Errorf("unsupported Terraform type %T", v)
	}
}

// tryUnpackPlistTaggedObject checks if an object's attributes represent a
// tagged plist type (__plist_type + value) and converts to the native Go type.
func tryUnpackPlistTaggedObject(attrs map[string]attr.Value) (interface{}, bool, error) {
	if len(attrs) != 2 {
		return nil, false, nil
	}

	typeAttr, hasType := attrs[plistTypeKey]
	valueAttr, hasValue := attrs[plistValueKey]
	if !hasType || !hasValue {
		return nil, false, nil
	}

	typeStr, ok := typeAttr.(basetypes.StringValue)
	if !ok || typeStr.IsNull() {
		return nil, false, nil
	}
	valueStr, ok := valueAttr.(basetypes.StringValue)
	if !ok || valueStr.IsNull() {
		return nil, false, nil
	}

	switch typeStr.ValueString() {
	case plistTypeDate:
		t, err := time.Parse(time.RFC3339, valueStr.ValueString())
		if err != nil {
			return nil, true, fmt.Errorf("invalid plist date %q: %w", valueStr.ValueString(), err)
		}
		return t, true, nil

	case plistTypeData:
		data, err := base64.StdEncoding.DecodeString(valueStr.ValueString())
		if err != nil {
			return nil, true, fmt.Errorf("invalid plist data (bad base64): %w", err)
		}
		return data, true, nil

	case plistTypeReal:
		f, err := strconv.ParseFloat(valueStr.ValueString(), 64)
		if err != nil {
			return nil, true, fmt.Errorf("invalid plist real %q: %w", valueStr.ValueString(), err)
		}
		return plistRealValue(f), true, nil

	default:
		return nil, false, nil
	}
}

// plistRealValue wraps a float64 to ensure it is encoded as <real> in plist,
// even if the value has no fractional part. goValueForPlistEncode passes it
// through without converting to int64. The plist library encodes it via
// MarshalPlist as a float64 → <real>.
type plistRealValue float64

func (r plistRealValue) MarshalPlist() (interface{}, error) {
	return float64(r), nil
}

// goValueForPlistEncode prepares a Go value for plist marshaling.
// It walks the structure and converts float64 values to int64 where appropriate,
// so the plist encoder produces <integer> vs <real> correctly.
func goValueForPlistEncode(v interface{}) interface{} {
	switch val := v.(type) {
	case float64:
		if val == math.Trunc(val) && !math.IsInf(val, 0) && !math.IsNaN(val) &&
			val >= math.MinInt64 && val <= math.MaxInt64 {
			return int64(val)
		}
		return val

	case []interface{}:
		result := make([]interface{}, len(val))
		for i, item := range val {
			result[i] = goValueForPlistEncode(item)
		}
		return result

	case map[string]interface{}:
		result := make(map[string]interface{}, len(val))
		for k, item := range val {
			result[k] = goValueForPlistEncode(item)
		}
		return result

	default:
		return val
	}
}

// goValueForBinaryEncode prepares a Go value for serializers (msgpack, CBOR) that don't honor orderedMap's MarshalJSON shortcut. Whole-number float64s are converted to int64 so encoded payloads use the integer family rather than IEEE-754 floats; map[string]interface{} is preserved (the encoder is responsible for sorting keys when determinism is required).
func goValueForBinaryEncode(v interface{}) interface{} {
	switch val := v.(type) {
	case float64:
		if val == math.Trunc(val) && !math.IsInf(val, 0) && !math.IsNaN(val) &&
			val >= math.MinInt64 && val <= math.MaxInt64 {
			return int64(val)
		}
		return val

	case []interface{}:
		result := make([]interface{}, len(val))
		for i, item := range val {
			result[i] = goValueForBinaryEncode(item)
		}
		return result

	case map[string]interface{}:
		result := make(map[string]interface{}, len(val))
		for k, item := range val {
			result[k] = goValueForBinaryEncode(item)
		}
		return result

	default:
		return val
	}
}

// goValueForJSONEncode prepares a Go value for JSON marshaling.
// It ensures stable key ordering by using sorted maps.
func goValueForJSONEncode(v interface{}) interface{} {
	switch val := v.(type) {
	case float64:
		if val == math.Trunc(val) && !math.IsInf(val, 0) && !math.IsNaN(val) &&
			val >= -1<<53 && val <= 1<<53 {
			return int64(val)
		}
		return val

	case []interface{}:
		result := make([]interface{}, len(val))
		for i, item := range val {
			result[i] = goValueForJSONEncode(item)
		}
		return result

	case map[string]interface{}:
		result := make(orderedMap, 0, len(val))
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			result = append(result, mapEntry{Key: k, Value: goValueForJSONEncode(val[k])})
		}
		return result

	default:
		return val
	}
}

// orderedMap is a JSON-marshalable ordered map that preserves key order.
type orderedMap []mapEntry

type mapEntry struct {
	Key   string
	Value interface{}
}

func (o orderedMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, entry := range o {
		if i > 0 {
			buf.WriteByte(',')
		}
		key, err := json.Marshal(entry.Key)
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteByte(':')
		val, err := json.Marshal(entry.Value)
		if err != nil {
			return nil, err
		}
		buf.Write(val)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// parseOptionsIndent extracts the "indent" string from a dynamic options value.
// Returns "" if no indent key is present (caller should use default).
func parseOptionsIndent(opts types.Dynamic) (string, error) {
	obj, ok := opts.UnderlyingValue().(basetypes.ObjectValue)
	if !ok {
		return "", fmt.Errorf("options must be an object, got %T", opts.UnderlyingValue())
	}
	return getStringOption(obj.Attributes(), "indent")
}

// getStringOption extracts an optional string value from an attributes map.
// Returns "" if the key is not present.
func getStringOption(attrs map[string]attr.Value, key string) (string, error) {
	v, ok := attrs[key]
	if !ok {
		return "", nil
	}
	sv, ok := v.(basetypes.StringValue)
	if !ok {
		return "", fmt.Errorf("%q must be a string, got %T", key, v)
	}
	return sv.ValueString(), nil
}

// getBoolOption extracts an optional bool value from an attributes map.
// Returns (false, false, nil) if the key is not present.
func getBoolOption(attrs map[string]attr.Value, key string) (value, present bool, err error) {
	v, ok := attrs[key]
	if !ok {
		return false, false, nil
	}
	bv, ok := v.(basetypes.BoolValue)
	if !ok {
		return false, false, fmt.Errorf("%q must be a bool, got %T", key, v)
	}
	return bv.ValueBool(), true, nil
}
