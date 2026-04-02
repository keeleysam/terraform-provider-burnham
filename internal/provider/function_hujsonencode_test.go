package provider

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runHuJSONEncode(t *testing.T, value attr.Value, indentArgs ...string) (string, *function.FuncError) {
	t.Helper()
	f := &HuJSONEncodeFunction{}

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

func TestHuJSONEncode_LargeObject(t *testing.T) {
	// Use a structure large enough that hujson expands it to multi-line.
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"name":        types.StringType,
			"enabled":     types.BoolType,
			"description": types.StringType,
			"version":     types.NumberType,
		},
		map[string]attr.Value{
			"name":        types.StringValue("test-profile"),
			"enabled":     types.BoolValue(true),
			"description": types.StringValue("A longer description for testing"),
			"version":     types.NumberValue(big.NewFloat(42)),
		},
	)

	result, err := runHuJSONEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Multi-line expanded objects get trailing commas.
	if !strings.Contains(result, ",\n") {
		t.Errorf("expected trailing commas in multi-line output:\n%s", result)
	}

	// Should be tab-indented by default.
	if !strings.Contains(result, "\t") {
		t.Errorf("expected tab indentation in output:\n%s", result)
	}

	// Should not contain the injected comment.
	if strings.Contains(result, "//") {
		t.Errorf("should not contain injected comment:\n%s", result)
	}
}

func TestHuJSONEncode_SmallObject(t *testing.T) {
	// Small objects stay on one line per hujson formatting (this is correct).
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"a": types.NumberType,
		},
		map[string]attr.Value{
			"a": types.NumberValue(big.NewFloat(1)),
		},
	)

	result, err := runHuJSONEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "{\"a\": 1}\n" {
		t.Errorf("expected compact single-line output, got:\n%q", result)
	}
}

func TestHuJSONEncode_CustomIndent(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"name":        types.StringType,
			"enabled":     types.BoolType,
			"description": types.StringType,
			"version":     types.NumberType,
		},
		map[string]attr.Value{
			"name":        types.StringValue("test-profile"),
			"enabled":     types.BoolValue(true),
			"description": types.StringValue("A longer description for testing"),
			"version":     types.NumberValue(big.NewFloat(42)),
		},
	)

	result, err := runHuJSONEncode(t, obj, "  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(result, "\t") {
		t.Errorf("expected no tabs in output with 2-space indent:\n%s", result)
	}
	if !strings.Contains(result, "  ") {
		t.Errorf("expected 2-space indentation in output:\n%s", result)
	}
}

func TestHuJSONEncode_NestedArray(t *testing.T) {
	// Build a nested structure with enough content to be expanded.
	innerArr := types.TupleValueMust(
		[]attr.Type{types.StringType, types.StringType, types.StringType},
		[]attr.Value{
			types.StringValue("user1@example.com"),
			types.StringValue("user2@example.com"),
			types.StringValue("user3@example.com"),
		},
	)

	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"members": innerArr.Type(nil),
		},
		map[string]attr.Value{
			"members": innerArr,
		},
	)

	result, err := runHuJSONEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's valid output (contains the values).
	if !strings.Contains(result, "user1@example.com") {
		t.Errorf("expected user1 in output:\n%s", result)
	}
}

func TestHuJSONEncode_NoInjectedComment(t *testing.T) {
	// Verify the output never contains the injected comment used internally.
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"key": types.StringType,
		},
		map[string]attr.Value{
			"key": types.StringValue("value"),
		},
	)

	result, err := runHuJSONEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.HasPrefix(result, "//") {
		t.Errorf("output should not start with injected comment:\n%s", result)
	}
}

func TestHuJSONEncode_TooManyIndentArgs(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{},
		map[string]attr.Value{},
	)
	_, err := runHuJSONEncode(t, obj, "  ", "\t")
	if err == nil {
		t.Fatal("expected error for too many indent args")
	}
}
