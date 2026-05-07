package transform

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestTerraformToJSON_Nil(t *testing.T) {
	got, err := terraformToJSON(types.DynamicNull())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestTerraformToJSON_Primitives(t *testing.T) {
	cases := []struct {
		name string
		in   attr.Value
		want interface{}
	}{
		{"bool true", types.BoolValue(true), true},
		{"bool false", types.BoolValue(false), false},
		{"string", types.StringValue("hi"), "hi"},
		{"empty string", types.StringValue(""), ""},
		{"int", types.NumberValue(big.NewFloat(7)), json.Number("7")},
		{"negative int", types.NumberValue(big.NewFloat(-3)), json.Number("-3")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := terraformToJSON(tc.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("want %v (%T), got %v (%T)", tc.want, tc.want, got, got)
			}
		})
	}
}

func TestTerraformToJSON_NumberFractional(t *testing.T) {
	got, err := terraformToJSON(types.NumberValue(big.NewFloat(1.5)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	n, ok := got.(json.Number)
	if !ok {
		t.Fatalf("expected json.Number, got %T", got)
	}
	if string(n) != "1.5" {
		t.Errorf("expected '1.5', got %q", string(n))
	}
}

func TestTerraformToJSON_NumberSmallMagnitude(t *testing.T) {
	// 'g' formatting would render 0.0000001 as "1e-07"; we use 'f' to keep decimal form.
	got, err := terraformToJSON(types.NumberValue(big.NewFloat(0.0000001)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	n, ok := got.(json.Number)
	if !ok {
		t.Fatalf("expected json.Number, got %T", got)
	}
	if string(n) == "1e-07" {
		t.Errorf("expected decimal form, got scientific %q", string(n))
	}
}

func TestTerraformToJSON_Tuple(t *testing.T) {
	tuple := types.TupleValueMust(
		[]attr.Type{types.StringType, types.NumberType},
		[]attr.Value{types.StringValue("a"), types.NumberValue(big.NewFloat(2))},
	)
	got, err := terraformToJSON(tuple)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	slice, ok := got.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", got)
	}
	if len(slice) != 2 || slice[0] != "a" || slice[1] != json.Number("2") {
		t.Errorf("unexpected tuple decode: %v", slice)
	}
}

func TestTerraformToJSON_Object(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{"x": types.StringType, "y": types.BoolType},
		map[string]attr.Value{"x": types.StringValue("v"), "y": types.BoolValue(true)},
	)
	got, err := terraformToJSON(obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := got.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{}, got %T", got)
	}
	if m["x"] != "v" || m["y"] != true {
		t.Errorf("unexpected object decode: %v", m)
	}
}

func TestJSONToTerraform_Nil(t *testing.T) {
	got, err := jsonToTerraform(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.IsNull() {
		t.Errorf("expected null, got %v", got)
	}
}

func TestJSONToTerraform_Primitives(t *testing.T) {
	cases := []struct {
		name string
		in   interface{}
	}{
		{"bool", true},
		{"string", "hi"},
		{"int", 7},
		{"int64", int64(-3)},
		{"float64", 1.5},
		{"json.Number int", json.Number("42")},
		{"json.Number frac", json.Number("3.14")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := jsonToTerraform(tc.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got == nil {
				t.Fatal("expected a value, got nil")
			}
		})
	}
}

func TestJSONToTerraform_NestedRoundtrip(t *testing.T) {
	original := map[string]interface{}{
		"name": "alice",
		"items": []interface{}{
			json.Number("1"),
			json.Number("2"),
			"three",
		},
		"nested": map[string]interface{}{
			"on": true,
		},
	}
	tf, err := jsonToTerraform(original)
	if err != nil {
		t.Fatalf("jsonToTerraform: %v", err)
	}
	back, err := terraformToJSON(tf)
	if err != nil {
		t.Fatalf("terraformToJSON: %v", err)
	}
	got, ok := back.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", back)
	}
	if got["name"] != "alice" {
		t.Errorf("name lost: %v", got["name"])
	}
	if items, ok := got["items"].([]interface{}); !ok || len(items) != 3 {
		t.Errorf("items lost: %v", got["items"])
	}
	if nested, ok := got["nested"].(map[string]interface{}); !ok || nested["on"] != true {
		t.Errorf("nested lost: %v", got["nested"])
	}
}

func TestJSONToTerraform_NaNRejected(t *testing.T) {
	_, err := jsonToTerraform(nanFloat())
	if err == nil {
		t.Error("expected error for NaN, got nil")
	}
}

func nanFloat() float64 {
	zero := 0.0
	return zero / zero
}

func TestBigFloatToJSONNumber(t *testing.T) {
	cases := []struct {
		in   *big.Float
		want string
	}{
		{big.NewFloat(0), "0"},
		{big.NewFloat(7), "7"},
		{big.NewFloat(-3), "-3"},
		{big.NewFloat(1.5), "1.5"},
	}
	for _, tc := range cases {
		got := bigFloatToJSONNumber(tc.in)
		if string(got) != tc.want {
			t.Errorf("input %v: want %q, got %q", tc.in, tc.want, string(got))
		}
	}
}
