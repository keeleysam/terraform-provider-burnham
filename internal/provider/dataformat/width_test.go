package dataformat

import (
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// widthFixture holds a short array that fits a modest width budget and a long
// one that does not, plus an array of objects. That combination exercises every
// decision the width option makes.
func widthFixture() attr.Value {
	shortElems := []attr.Type{types.StringType}
	longElems := make([]attr.Type, 6)
	longVals := make([]attr.Value, 6)
	for i := range longElems {
		longElems[i] = types.StringType
		longVals[i] = types.StringValue("a-reasonably-long-element-value-" + strings.Repeat("x", 8))
	}
	objType := types.ObjectType{AttrTypes: map[string]attr.Type{"k": types.StringType}}

	return types.ObjectValueMust(
		map[string]attr.Type{
			"short":   types.TupleType{ElemTypes: shortElems},
			"long":    types.TupleType{ElemTypes: longElems},
			"objects": types.TupleType{ElemTypes: []attr.Type{objType, objType}},
		},
		map[string]attr.Value{
			"short": types.TupleValueMust(shortElems, []attr.Value{types.StringValue("*")}),
			"long":  types.TupleValueMust(longElems, longVals),
			"objects": types.TupleValueMust([]attr.Type{objType, objType}, []attr.Value{
				types.ObjectValueMust(map[string]attr.Type{"k": types.StringType}, map[string]attr.Value{"k": types.StringValue("one")}),
				types.ObjectValueMust(map[string]attr.Type{"k": types.StringType}, map[string]attr.Value{"k": types.StringValue("two")}),
			}),
		},
	)
}

func numOpts(key string, n int) attr.Value {
	return types.ObjectValueMust(
		map[string]attr.Type{key: types.NumberType},
		map[string]attr.Value{key: types.NumberValue(big.NewFloat(float64(n)))},
	)
}

func TestJSONEncode_WidthUnsetIsUnchanged(t *testing.T) {
	withoutOpts, err := runJSONEncode(t, widthFixture())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Every element on its own line means no element shares a line with a bracket.
	if strings.Contains(withoutOpts, `["`) {
		t.Errorf("expected fully expanded output, got a packed array:\n%s", withoutOpts)
	}
}

func TestJSONEncode_WidthPacksShortArraysOnly(t *testing.T) {
	got, err := runJSONEncode(t, widthFixture(), numOpts("width", 78))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, `"short": ["*"]`) {
		t.Errorf("short array should be packed onto one line:\n%s", got)
	}
	if strings.Contains(got, `"long": ["a-reasonably`) {
		t.Errorf("long array should not be packed, it exceeds the width budget:\n%s", got)
	}

	// The shapes that motivated the option: an object must never share a line
	// with a sibling object or with an enclosing bracket.
	for _, bad := range []string{"}, {", "[{", "}]"} {
		if strings.Contains(got, bad) {
			t.Errorf("output contains %q, objects must never pack:\n%s", bad, got)
		}
	}

	// Whatever the layout, jsonencode must still emit valid JSON so Terraform's
	// plan renderer gives it a structural diff.
	if !json.Valid([]byte(got)) {
		t.Errorf("output is not valid JSON:\n%s", got)
	}
}

func TestJSONEncode_WidthRejectsNegative(t *testing.T) {
	_, err := runJSONEncode(t, widthFixture(), numOpts("width", -1))
	if err == nil {
		t.Fatal("expected an error for a negative width")
	}
	if !strings.Contains(err.Error(), "must not be negative") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHuJSONEncode_WidthKeepsTrailingCommas(t *testing.T) {
	got, err := runHuJSONEncode(t, widthFixture(), numOpts("width", 78))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// hujsonencode always emits trailing commas; there is no option to disable
	// them. A caller who wants standard JSON wants jsonencode.
	if !strings.Contains(got, ",\n") || json.Valid([]byte(got)) {
		t.Errorf("expected HuJSON with trailing commas, which is not valid JSON:\n%s", got)
	}
	if strings.Contains(got, "}, {") || strings.Contains(got, "[{") {
		t.Errorf("objects must never pack:\n%s", got)
	}
	if !strings.Contains(got, `"short": ["*"`) {
		t.Errorf("short array should be packed onto one line:\n%s", got)
	}
}

func TestHuJSONEncode_WidthConflictsWithCompact(t *testing.T) {
	opts := types.ObjectValueMust(
		map[string]attr.Type{"width": types.NumberType, "compact": types.BoolType},
		map[string]attr.Value{"width": types.NumberValue(big.NewFloat(78)), "compact": types.BoolValue(true)},
	)
	_, err := runHuJSONEncode(t, widthFixture(), opts)
	if err == nil {
		t.Fatal("expected an error when both width and compact are set")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWidthMustBeWholeNumber(t *testing.T) {
	opts := types.ObjectValueMust(
		map[string]attr.Type{"width": types.NumberType},
		map[string]attr.Value{"width": types.NumberValue(big.NewFloat(78.5))},
	)
	_, err := runJSONEncode(t, widthFixture(), opts)
	if err == nil {
		t.Fatal("expected an error for a fractional width")
	}
	if !strings.Contains(err.Error(), "whole number") {
		t.Errorf("unexpected error: %v", err)
	}
}
