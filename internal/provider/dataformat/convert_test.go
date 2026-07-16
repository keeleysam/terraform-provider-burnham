package dataformat

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestGoToTerraformValue_Nil(t *testing.T) {
	val, err := goToTerraformValue(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !val.IsNull() {
		t.Errorf("expected null, got %v", val)
	}
}

func TestGoToTerraformValue_Bool(t *testing.T) {
	val, err := goToTerraformValue(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	bv, ok := val.(types.Bool)
	if !ok {
		t.Fatalf("expected Bool, got %T", val)
	}
	if !bv.ValueBool() {
		t.Error("expected true")
	}
}

func TestGoToTerraformValue_String(t *testing.T) {
	val, err := goToTerraformValue("hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sv, ok := val.(types.String)
	if !ok {
		t.Fatalf("expected String, got %T", val)
	}
	if sv.ValueString() != "hello" {
		t.Errorf("expected 'hello', got %q", sv.ValueString())
	}
}

func TestGoToTerraformValue_JSONNumber(t *testing.T) {
	val, err := goToTerraformValue(json.Number("42"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	nv, ok := val.(types.Number)
	if !ok {
		t.Fatalf("expected Number, got %T", val)
	}
	f, _ := nv.ValueBigFloat().Float64()
	if f != 42 {
		t.Errorf("expected 42, got %f", f)
	}
}

func TestGoToTerraformValue_JSONNumberInvalid(t *testing.T) {
	_, err := goToTerraformValue(json.Number("notanumber"))
	if err == nil {
		t.Fatal("expected error for invalid json.Number")
	}
}

func TestGoToTerraformValue_UnsupportedType(t *testing.T) {
	_, err := goToTerraformValue(struct{}{})
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestGoToTerraformValue_WholeFloat64TaggedReal(t *testing.T) {
	// Whole-number float64 should produce a tagged real object in plist mode.
	val, err := goToTerraformValuePlist(float64(2))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	obj, ok := val.(types.Object)
	if !ok {
		t.Fatalf("expected Object (tagged real), got %T", val)
	}
	attrs := obj.Attributes()
	typeVal := attrs[plistTypeKey].(types.String).ValueString()
	if typeVal != plistTypeReal {
		t.Errorf("expected __plist_type=%q, got %q", plistTypeReal, typeVal)
	}
}

func TestGoToTerraformValue_FractionalFloat64PlainNumber(t *testing.T) {
	// Fractional float64 should produce a plain number.
	val, err := goToTerraformValue(float64(3.14))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, ok := val.(types.Number)
	if !ok {
		t.Fatalf("expected Number, got %T", val)
	}
}

func TestGoToTerraformValue_Numbers(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected float64
	}{
		{"int", 42, 42},
		{"int64", int64(100), 100},
		{"uint64", uint64(999), 999},
		{"float32", float32(1.5), 1.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := goToTerraformValue(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			nv, ok := val.(types.Number)
			if !ok {
				t.Fatalf("expected Number, got %T", val)
			}
			f, _ := nv.ValueBigFloat().Float64()
			if f != tt.expected {
				t.Errorf("expected %f, got %f", tt.expected, f)
			}
		})
	}
}

func TestGoToTerraformValue_TimeTaggedObject(t *testing.T) {
	ts := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	val, err := goToTerraformValuePlist(ts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	obj, ok := val.(types.Object)
	if !ok {
		t.Fatalf("expected Object, got %T", val)
	}
	attrs := obj.Attributes()
	typeVal := attrs[plistTypeKey].(types.String).ValueString()
	valueVal := attrs[plistValueKey].(types.String).ValueString()
	if typeVal != plistTypeDate {
		t.Errorf("expected __plist_type=%q, got %q", plistTypeDate, typeVal)
	}
	if valueVal != "2025-06-01T00:00:00Z" {
		t.Errorf("expected value=%q, got %q", "2025-06-01T00:00:00Z", valueVal)
	}
}

func TestGoToTerraformValue_BytesTaggedObject(t *testing.T) {
	data := []byte("hello world")
	val, err := goToTerraformValuePlist(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	obj, ok := val.(types.Object)
	if !ok {
		t.Fatalf("expected Object, got %T", val)
	}
	attrs := obj.Attributes()
	typeVal := attrs[plistTypeKey].(types.String).ValueString()
	valueVal := attrs[plistValueKey].(types.String).ValueString()
	if typeVal != plistTypeData {
		t.Errorf("expected __plist_type=%q, got %q", plistTypeData, typeVal)
	}
	expected := base64.StdEncoding.EncodeToString(data)
	if valueVal != expected {
		t.Errorf("expected value=%q, got %q", expected, valueVal)
	}
}

func TestGoToTerraformValue_Slice(t *testing.T) {
	input := []interface{}{"a", float64(1), true}
	val, err := goToTerraformValue(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tv, ok := val.(types.Tuple)
	if !ok {
		t.Fatalf("expected Tuple, got %T", val)
	}
	elems := tv.Elements()
	if len(elems) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(elems))
	}
}

func TestGoToTerraformValue_Map(t *testing.T) {
	input := map[string]interface{}{
		"name":  "test",
		"count": float64(5),
	}
	val, err := goToTerraformValue(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	obj, ok := val.(types.Object)
	if !ok {
		t.Fatalf("expected Object, got %T", val)
	}
	attrs := obj.Attributes()
	if len(attrs) != 2 {
		t.Fatalf("expected 2 attributes, got %d", len(attrs))
	}
}

func TestGoToTerraformValue_EmptySlice(t *testing.T) {
	val, err := goToTerraformValue([]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tv, ok := val.(types.Tuple)
	if !ok {
		t.Fatalf("expected Tuple, got %T", val)
	}
	if len(tv.Elements()) != 0 {
		t.Errorf("expected empty tuple, got %d elements", len(tv.Elements()))
	}
}

func TestGoToTerraformValue_EmptyMap(t *testing.T) {
	val, err := goToTerraformValue(map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	obj, ok := val.(types.Object)
	if !ok {
		t.Fatalf("expected Object, got %T", val)
	}
	if len(obj.Attributes()) != 0 {
		t.Errorf("expected empty object, got %d attributes", len(obj.Attributes()))
	}
}

func TestTerraformValueToGo_Primitives(t *testing.T) {
	tests := []struct {
		name     string
		input    attr.Value
		expected interface{}
	}{
		{"bool", types.BoolValue(true), true},
		{"string", types.StringValue("hello"), "hello"},
		{"number", types.NumberValue(big.NewFloat(42)), int64(42)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := terraformValueToGo(tt.input, false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v (%T), got %v (%T)", tt.expected, tt.expected, result, result)
			}
		})
	}
}

func TestTerraformValueToGo_Null(t *testing.T) {
	result, err := terraformValueToGo(types.StringNull(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestTerraformValueToGo_PlistTaggedDate(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			plistTypeKey:  types.StringType,
			plistValueKey: types.StringType,
		},
		map[string]attr.Value{
			plistTypeKey:  types.StringValue(plistTypeDate),
			plistValueKey: types.StringValue("2025-06-01T00:00:00Z"),
		},
	)

	result, err := terraformValueToGo(obj, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ts, ok := result.(time.Time)
	if !ok {
		t.Fatalf("expected time.Time, got %T", result)
	}
	expected := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	if !ts.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, ts)
	}
}

func TestTerraformValueToGo_PlistTaggedData(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("binary data"))
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			plistTypeKey:  types.StringType,
			plistValueKey: types.StringType,
		},
		map[string]attr.Value{
			plistTypeKey:  types.StringValue(plistTypeData),
			plistValueKey: types.StringValue(encoded),
		},
	)

	result, err := terraformValueToGo(obj, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, ok := result.([]byte)
	if !ok {
		t.Fatalf("expected []byte, got %T", result)
	}
	if string(data) != "binary data" {
		t.Errorf("expected 'binary data', got %q", string(data))
	}
}

func TestTerraformValueToGo_PlistModeOff(t *testing.T) {
	// With plistMode=false, tagged objects should be treated as normal objects.
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			plistTypeKey:  types.StringType,
			plistValueKey: types.StringType,
		},
		map[string]attr.Value{
			plistTypeKey:  types.StringValue(plistTypeDate),
			plistValueKey: types.StringValue("2025-06-01T00:00:00Z"),
		},
	)

	result, err := terraformValueToGo(obj, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if m[plistTypeKey] != plistTypeDate {
		t.Errorf("expected __plist_type=date in map, got %v", m)
	}
}

func TestGoValueForPlistEncode_IntegerConversion(t *testing.T) {
	result := goValueForPlistEncode(float64(42))
	if v, ok := result.(int64); !ok || v != 42 {
		t.Errorf("expected int64(42), got %v (%T)", result, result)
	}

	result = goValueForPlistEncode(float64(3.14))
	if v, ok := result.(float64); !ok || v != 3.14 {
		t.Errorf("expected float64(3.14), got %v (%T)", result, result)
	}
}

func TestGoValueForPlistEncode_Nested(t *testing.T) {
	input := map[string]interface{}{
		"count": float64(5),
		"items": []interface{}{float64(1), float64(2.5)},
	}
	result := goValueForPlistEncode(input)
	m := result.(map[string]interface{})
	if _, ok := m["count"].(int64); !ok {
		t.Errorf("expected count to be int64, got %T", m["count"])
	}
	items := m["items"].([]interface{})
	if _, ok := items[0].(int64); !ok {
		t.Errorf("expected items[0] to be int64, got %T", items[0])
	}
	if _, ok := items[1].(float64); !ok {
		t.Errorf("expected items[1] to be float64, got %T", items[1])
	}
}

func TestTerraformValueToGo_ListValue(t *testing.T) {
	list, _ := types.ListValue(types.StringType, []attr.Value{
		types.StringValue("a"),
		types.StringValue("b"),
	})
	result, err := terraformValueToGo(list, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	slice, ok := result.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", result)
	}
	if len(slice) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(slice))
	}
	if slice[0] != "a" || slice[1] != "b" {
		t.Errorf("expected [a, b], got %v", slice)
	}
}

func TestTerraformValueToGo_MapValue(t *testing.T) {
	m, _ := types.MapValue(types.StringType, map[string]attr.Value{
		"key1": types.StringValue("val1"),
		"key2": types.StringValue("val2"),
	})
	result, err := terraformValueToGo(m, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	goMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{}, got %T", result)
	}
	if goMap["key1"] != "val1" || goMap["key2"] != "val2" {
		t.Errorf("expected {key1: val1, key2: val2}, got %v", goMap)
	}
}

func TestTerraformValueToGo_SetValue(t *testing.T) {
	s, _ := types.SetValue(types.StringType, []attr.Value{
		types.StringValue("x"),
		types.StringValue("y"),
	})
	result, err := terraformValueToGo(s, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	slice, ok := result.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", result)
	}
	if len(slice) != 2 {
		t.Errorf("expected 2 elements, got %d", len(slice))
	}
}

func TestTerraformValueToGo_DynamicValue(t *testing.T) {
	dyn := types.DynamicValue(types.StringValue("wrapped"))
	result, err := terraformValueToGo(dyn, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "wrapped" {
		t.Errorf("expected 'wrapped', got %v", result)
	}
}

func TestTryUnpackPlistTaggedObject_WrongKeyCount(t *testing.T) {
	attrs := map[string]attr.Value{
		plistTypeKey:  types.StringValue(plistTypeDate),
		plistValueKey: types.StringValue("2025-06-01T00:00:00Z"),
		"extra":       types.StringValue("nope"),
	}
	_, ok, err := tryUnpackPlistTaggedObject(attrs)
	if ok || err != nil {
		t.Errorf("expected (nil, false, nil) for 3-key object, got ok=%v err=%v", ok, err)
	}
}

func TestTryUnpackPlistTaggedObject_UnknownType(t *testing.T) {
	attrs := map[string]attr.Value{
		plistTypeKey:  types.StringValue("unknown_type"),
		plistValueKey: types.StringValue("something"),
	}
	_, ok, err := tryUnpackPlistTaggedObject(attrs)
	if ok || err != nil {
		t.Errorf("expected (nil, false, nil) for unknown plist type, got ok=%v err=%v", ok, err)
	}
}

func TestTryUnpackPlistTaggedObject_MissingKeys(t *testing.T) {
	attrs := map[string]attr.Value{
		"wrong_key":  types.StringValue("date"),
		"also_wrong": types.StringValue("2025-06-01T00:00:00Z"),
	}
	_, ok, err := tryUnpackPlistTaggedObject(attrs)
	if ok || err != nil {
		t.Errorf("expected (nil, false, nil) for missing keys, got ok=%v err=%v", ok, err)
	}
}

func TestTryUnpackPlistTaggedObject_Real(t *testing.T) {
	attrs := map[string]attr.Value{
		plistTypeKey:  types.StringValue(plistTypeReal),
		plistValueKey: types.StringValue("2"),
	}
	result, ok, err := tryUnpackPlistTaggedObject(attrs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true for real type")
	}
	rv, rvOk := result.(plistRealValue)
	if !rvOk {
		t.Fatalf("expected plistRealValue, got %T", result)
	}
	if float64(rv) != 2.0 {
		t.Errorf("expected 2.0, got %f", float64(rv))
	}
}

func TestTryUnpackPlistTaggedObject_InvalidDate(t *testing.T) {
	attrs := map[string]attr.Value{
		plistTypeKey:  types.StringValue(plistTypeDate),
		plistValueKey: types.StringValue("not-a-date"),
	}
	_, _, err := tryUnpackPlistTaggedObject(attrs)
	if err == nil {
		t.Fatal("expected error for invalid date")
	}
}

func TestTryUnpackPlistTaggedObject_InvalidBase64(t *testing.T) {
	attrs := map[string]attr.Value{
		plistTypeKey:  types.StringValue(plistTypeData),
		plistValueKey: types.StringValue("!!!not-base64!!!"),
	}
	_, _, err := tryUnpackPlistTaggedObject(attrs)
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestTryUnpackPlistTaggedObject_InvalidReal(t *testing.T) {
	attrs := map[string]attr.Value{
		plistTypeKey:  types.StringValue(plistTypeReal),
		plistValueKey: types.StringValue("not-a-number"),
	}
	_, _, err := tryUnpackPlistTaggedObject(attrs)
	if err == nil {
		t.Fatal("expected error for invalid real")
	}
}

// makeExactNumber builds a NumberValue whose big.Float carries an exact integer
// parsed from its decimal digits, mirroring how Terraform hands the provider a
// high-precision numeric literal. big.NewFloat(<float literal>) would already
// have rounded the value to float64, so the digits have to come in through
// big.Int to reproduce integers beyond 2^53 faithfully.
func makeExactNumber(t *testing.T, digits string) types.Number {
	t.Helper()
	bi, ok := new(big.Int).SetString(digits, 10)
	if !ok {
		t.Fatalf("bad integer literal %q", digits)
	}
	return types.NumberValue(new(big.Float).SetInt(bi))
}

func runCBOREncodeForTest(t *testing.T, value attr.Value) string {
	t.Helper()
	f := &CBOREncodeFunction{}
	args := function.NewArgumentsData([]attr.Value{types.DynamicValue(value)})
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.StringValue(""))}
	f.Run(context.Background(), req, resp)
	if resp.Error != nil {
		t.Fatalf("cbor encode error: %v", resp.Error)
	}
	return resp.Result.Value().(types.String).ValueString()
}

// TestTerraformValueToGo_IntegerPrecision guards the encode path against
// collapsing an exact integer to float64. A big.Float carrying 2^53+1 (the
// smallest positive integer float64 cannot represent) must come back as an exact
// integer, not a rounded float.
func TestTerraformValueToGo_IntegerPrecision(t *testing.T) {
	got, err := terraformValueToGo(makeExactNumber(t, "9007199254740993"), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	i, ok := got.(int64)
	if !ok {
		t.Fatalf("expected int64, got %T", got)
	}
	if i != 9007199254740993 {
		t.Errorf("precision lost: want 9007199254740993, got %d", i)
	}
}

// TestTerraformValueToGo_IntegerBeyondInt64 checks that an integer too large for
// int64 is carried as a *big.Int rather than a lossy float64.
func TestTerraformValueToGo_IntegerBeyondInt64(t *testing.T) {
	got, err := terraformValueToGo(makeExactNumber(t, "12345678901234567890"), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	bi, ok := got.(*big.Int)
	if !ok {
		t.Fatalf("expected *big.Int, got %T (%v)", got, got)
	}
	if bi.String() != "12345678901234567890" {
		t.Errorf("precision lost: want 12345678901234567890, got %s", bi.String())
	}
}

func TestJSONEncode_IntegerPrecision(t *testing.T) {
	result, err := runJSONEncode(t, makeExactNumber(t, "9007199254740993"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "9007199254740993" {
		t.Errorf("expected 9007199254740993, got %q", result)
	}
}

func TestJSONEncode_IntegerBeyondInt64(t *testing.T) {
	result, err := runJSONEncode(t, makeExactNumber(t, "12345678901234567890"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "12345678901234567890" {
		t.Errorf("expected 12345678901234567890, got %q", result)
	}
}

func TestCBOREncode_IntegerRoundTrip(t *testing.T) {
	// This value exceeds uint64 and is not a power of two, so a lossy float64 hop
	// would corrupt it. Encode then decode and confirm the number survives
	// bit-for-bit.
	huge, ok := new(big.Int).SetString("12345678901234567890123", 10)
	if !ok {
		t.Fatal("bad literal")
	}
	encoded := runCBOREncodeForTest(t, types.NumberValue(new(big.Float).SetInt(huge)))

	got, ferr := runCBORDecode(t, encoded)
	if ferr != nil {
		t.Fatalf("decode error: %v", ferr)
	}
	n, ok := got.UnderlyingValue().(types.Number)
	if !ok {
		t.Fatalf("expected Number, got %T", got.UnderlyingValue())
	}
	if n.ValueBigFloat().Cmp(new(big.Float).SetInt(huge)) != 0 {
		t.Errorf("precision lost: want %s, got %s", huge.String(), n.ValueBigFloat().Text('f', -1))
	}
}

func TestPlistEncode_IntegerPrecision(t *testing.T) {
	result, err := runPlistEncode(t, makeExactNumber(t, "9007199254740993"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "<integer>9007199254740993</integer>") {
		t.Errorf("expected <integer>9007199254740993</integer> in output:\n%s", result)
	}
}

func TestKDLEncode_IntegerPrecision(t *testing.T) {
	node := makeKDLNode("n", []attr.Value{makeExactNumber(t, "9007199254740993")}, nil, nil)
	nodes := types.TupleValueMust([]attr.Type{node.Type(nil)}, []attr.Value{node})

	result, err := runKDLEncode(t, nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "9007199254740993") {
		t.Errorf("expected 9007199254740993 in output:\n%s", result)
	}
}

func TestGoValueForJSONEncode_WholeNumber(t *testing.T) {
	result := goValueForJSONEncode(float64(42))
	if v, ok := result.(int64); !ok || v != 42 {
		t.Errorf("expected int64(42), got %v (%T)", result, result)
	}
}

func TestGoValueForJSONEncode_Fractional(t *testing.T) {
	result := goValueForJSONEncode(float64(3.14))
	if v, ok := result.(float64); !ok || v != 3.14 {
		t.Errorf("expected float64(3.14), got %v (%T)", result, result)
	}
}

func TestGoValueForJSONEncode_SortedKeys(t *testing.T) {
	input := map[string]interface{}{
		"zebra": "z",
		"apple": "a",
		"mango": "m",
	}
	result := goValueForJSONEncode(input)
	om, ok := result.(orderedMap)
	if !ok {
		t.Fatalf("expected orderedMap, got %T", result)
	}
	if len(om) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(om))
	}
	if om[0].Key != "apple" || om[1].Key != "mango" || om[2].Key != "zebra" {
		t.Errorf("expected sorted keys [apple, mango, zebra], got [%s, %s, %s]", om[0].Key, om[1].Key, om[2].Key)
	}
}

func TestGoValueForJSONEncode_Nested(t *testing.T) {
	input := map[string]interface{}{
		"items": []interface{}{float64(1), float64(2.5)},
		"count": float64(10),
	}
	result := goValueForJSONEncode(input)
	om := result.(orderedMap)
	// "count" sorts before "items"
	if om[0].Key != "count" {
		t.Errorf("expected first key 'count', got %q", om[0].Key)
	}
	if v, ok := om[0].Value.(int64); !ok || v != 10 {
		t.Errorf("expected count=int64(10), got %v (%T)", om[0].Value, om[0].Value)
	}
}
