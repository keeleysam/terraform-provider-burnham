package dataformat

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runJSONCanonicalize(t *testing.T, value attr.Value) (string, *function.FuncError) {
	t.Helper()
	f := &JSONCanonicalizeFunction{}
	args := function.NewArgumentsData([]attr.Value{types.DynamicValue(value)})
	resp := &function.RunResponse{Result: function.NewResultData(types.StringValue(""))}
	f.Run(context.Background(), function.RunRequest{Arguments: args}, resp)
	if resp.Error != nil {
		return "", resp.Error
	}
	return resp.Result.Value().(types.String).ValueString(), nil
}

// fromJSON builds a Terraform value from a JSON literal, so tests can express inputs compactly.
func fromJSON(t *testing.T, s string) attr.Value {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	var raw interface{}
	if err := dec.Decode(&raw); err != nil {
		t.Fatalf("parse input JSON: %v", err)
	}
	v, err := goToTerraformValue(raw)
	if err != nil {
		t.Fatalf("build value: %v", err)
	}
	return v
}

// TestJSONCanonicalize_KeyOrdering asserts RFC 8785 lexicographic member ordering and whitespace removal.
func TestJSONCanonicalize_KeyOrdering(t *testing.T) {
	got, ferr := runJSONCanonicalize(t, fromJSON(t, `{ "c": 3, "a": 1, "b": 2 }`))
	if ferr != nil {
		t.Fatalf("canonicalize: %v", ferr)
	}
	want := `{"a":1,"b":2,"c":3}`
	if got != want {
		t.Fatalf("ordering:\n got %s\nwant %s", got, want)
	}
}

// TestJSONCanonicalize_Numbers asserts RFC 8785 ES6 number formatting on representative values.
func TestJSONCanonicalize_Numbers(t *testing.T) {
	got, ferr := runJSONCanonicalize(t, fromJSON(t, `{"a":4.50,"b":100,"c":0.002,"d":1e30,"e":2e-3}`))
	if ferr != nil {
		t.Fatalf("canonicalize: %v", ferr)
	}
	want := `{"a":4.5,"b":100,"c":0.002,"d":1e+30,"e":0.002}`
	if got != want {
		t.Fatalf("numbers:\n got %s\nwant %s", got, want)
	}
}

// TestJSONCanonicalize_NestedAndArrays checks nesting, arrays, and literal preservation.
func TestJSONCanonicalize_NestedAndArrays(t *testing.T) {
	got, ferr := runJSONCanonicalize(t, fromJSON(t, `{"z":[3,2,1],"y":{"b":false,"a":true},"x":null}`))
	if ferr != nil {
		t.Fatalf("canonicalize: %v", ferr)
	}
	want := `{"x":null,"y":{"a":true,"b":false},"z":[3,2,1]}`
	if got != want {
		t.Fatalf("nested:\n got %s\nwant %s", got, want)
	}
}

// TestJSONCanonicalize_UnicodeAndEscapes checks minimal canonical string escaping: short escapes for control chars, literal non-ASCII.
func TestJSONCanonicalize_UnicodeAndEscapes(t *testing.T) {
	// Build a string value directly to include a real control character (tab) and a non-ASCII rune.
	val := types.ObjectValueMust(
		map[string]attr.Type{"s": types.StringType},
		map[string]attr.Value{"s": types.StringValue("a\tb€")},
	)
	got, ferr := runJSONCanonicalize(t, val)
	if ferr != nil {
		t.Fatalf("canonicalize: %v", ferr)
	}
	want := "{\"s\":\"a\\tb\\u0001€\"}"
	if got != want {
		t.Fatalf("unicode:\n got %q\nwant %q", got, want)
	}
}

// TestJSONCanonicalize_LargeIntegerFollowsDouble documents that integers beyond 2^53 follow RFC 8785 double serialization.
func TestJSONCanonicalize_LargeIntegerFollowsDouble(t *testing.T) {
	big1 := new(big.Int)
	big1.SetString("9223372036854775807", 10) // 2^63 - 1
	val := types.ObjectValueMust(
		map[string]attr.Type{"n": types.NumberType},
		map[string]attr.Value{"n": types.NumberValue(new(big.Float).SetInt(big1))},
	)
	got, ferr := runJSONCanonicalize(t, val)
	if ferr != nil {
		t.Fatalf("canonicalize: %v", ferr)
	}
	// 2^63-1 has no exact IEEE-754 double; ES6 serialization rounds to 9223372036854776000.
	want := `{"n":9223372036854776000}`
	if got != want {
		t.Fatalf("large int:\n got %s\nwant %s", got, want)
	}
}

// TestJSONCanonicalize_Deterministic confirms byte-identical output across two calls.
func TestJSONCanonicalize_Deterministic(t *testing.T) {
	in := fromJSON(t, `{"b":2,"a":[1,2,{"d":4,"c":3}]}`)
	a, _ := runJSONCanonicalize(t, in)
	b, _ := runJSONCanonicalize(t, in)
	if a != b {
		t.Fatalf("non-deterministic: %q vs %q", a, b)
	}
}

func TestJSONCanonicalize_UnknownYieldsUnknown(t *testing.T) {
	f := &JSONCanonicalizeFunction{}
	args := function.NewArgumentsData([]attr.Value{types.DynamicUnknown()})
	resp := &function.RunResponse{Result: function.NewResultData(types.StringValue(""))}
	f.Run(context.Background(), function.RunRequest{Arguments: args}, resp)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	if !resp.Result.Value().(types.String).IsUnknown() {
		t.Fatal("expected unknown result")
	}
}
