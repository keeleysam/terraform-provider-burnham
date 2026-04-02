package provider

import (
	"context"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runJSONEncode(t *testing.T, value attr.Value, indentArgs ...string) (string, *function.FuncError) {
	t.Helper()
	f := &JSONEncodeFunction{}

	// Build the variadic tuple for indent args.
	indentElems := make([]attr.Value, len(indentArgs))
	indentTypes := make([]attr.Type, len(indentArgs))
	for i, s := range indentArgs {
		indentElems[i] = types.StringValue(s)
		indentTypes[i] = types.StringType
	}
	variadicTuple := types.TupleValueMust(indentTypes, indentElems)

	args := function.NewArgumentsData([]attr.Value{
		types.DynamicValue(value),
		variadicTuple,
	})

	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.StringValue(""))}

	f.Run(context.Background(), req, resp)

	if resp.Error != nil {
		return "", resp.Error
	}

	result, ok := resp.Result.Value().(types.String)
	if !ok {
		t.Fatalf("expected String result, got %T", resp.Result.Value())
	}
	return result.ValueString(), nil
}

func TestJSONEncode_SimpleObject(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"a": types.NumberType,
			"b": types.BoolType,
		},
		map[string]attr.Value{
			"a": types.NumberValue(big.NewFloat(1)),
			"b": types.BoolValue(true),
		},
	)

	result, err := runJSONEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "{\n\t\"a\": 1,\n\t\"b\": true\n}"
	if result != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, result)
	}
}

func TestJSONEncode_TwoSpaces(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"a": types.NumberType,
		},
		map[string]attr.Value{
			"a": types.NumberValue(big.NewFloat(1)),
		},
	)

	result, err := runJSONEncode(t, obj, "  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "{\n  \"a\": 1\n}"
	if result != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, result)
	}
}

func TestJSONEncode_TooManyIndentArgs(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{},
		map[string]attr.Value{},
	)
	_, err := runJSONEncode(t, obj, "  ", "\t")
	if err == nil {
		t.Fatal("expected error for too many indent args")
	}
}

func TestJSONEncode_String(t *testing.T) {
	result, err := runJSONEncode(t, types.StringValue("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "\"hello\"" {
		t.Errorf("expected '\"hello\"', got %q", result)
	}
}

func TestJSONEncode_Number(t *testing.T) {
	result, err := runJSONEncode(t, types.NumberValue(big.NewFloat(42)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "42" {
		t.Errorf("expected '42', got %q", result)
	}
}

func TestJSONEncode_NestedStructure(t *testing.T) {
	inner := types.TupleValueMust(
		[]attr.Type{types.NumberType, types.NumberType},
		[]attr.Value{
			types.NumberValue(big.NewFloat(1)),
			types.NumberValue(big.NewFloat(2)),
		},
	)

	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"items": inner.Type(nil),
		},
		map[string]attr.Value{
			"items": inner,
		},
	)

	result, err := runJSONEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "{\n\t\"items\": [\n\t\t1,\n\t\t2\n\t]\n}"
	if result != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, result)
	}
}
