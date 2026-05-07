package dataformat

import (
	"context"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runNDJSONDecode(t *testing.T, input string) (types.Dynamic, *function.FuncError) {
	t.Helper()
	f := &NDJSONDecodeFunction{}
	args := function.NewArgumentsData([]attr.Value{types.StringValue(input)})
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.DynamicNull())}
	f.Run(context.Background(), req, resp)
	if resp.Error != nil {
		return types.DynamicNull(), resp.Error
	}
	return resp.Result.Value().(types.Dynamic), nil
}

func runNDJSONEncode(t *testing.T, value attr.Value) (string, *function.FuncError) {
	t.Helper()
	f := &NDJSONEncodeFunction{}
	args := function.NewArgumentsData([]attr.Value{types.DynamicValue(value)})
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.StringValue(""))}
	f.Run(context.Background(), req, resp)
	if resp.Error != nil {
		return "", resp.Error
	}
	return resp.Result.Value().(types.String).ValueString(), nil
}

func TestNDJSONDecode_BlankLinesTolerated(t *testing.T) {
	got, err := runNDJSONDecode(t, "{\"a\":1}\n\n\n{\"a\":2}\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tup := got.UnderlyingValue().(types.Tuple)
	if len(tup.Elements()) != 2 {
		t.Errorf("want 2 elements, got %d", len(tup.Elements()))
	}
}

func TestNDJSONDecode_Empty(t *testing.T) {
	got, err := runNDJSONDecode(t, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tup := got.UnderlyingValue().(types.Tuple)
	if len(tup.Elements()) != 0 {
		t.Errorf("want 0 elements, got %d", len(tup.Elements()))
	}
}

func TestNDJSONDecode_MixedTypes(t *testing.T) {
	got, err := runNDJSONDecode(t, "1\n\"hello\"\n[1,2]\nnull\n{\"k\":true}\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tup := got.UnderlyingValue().(types.Tuple)
	if len(tup.Elements()) != 5 {
		t.Errorf("want 5 elements, got %d", len(tup.Elements()))
	}
}

func TestNDJSONDecode_Malformed(t *testing.T) {
	_, err := runNDJSONDecode(t, "{not json}")
	if err == nil {
		t.Error("expected error for malformed input")
	}
}

func TestNDJSONEncode_TrailingNewline(t *testing.T) {
	in := types.TupleValueMust(
		[]attr.Type{types.ObjectType{AttrTypes: map[string]attr.Type{"a": types.NumberType}}},
		[]attr.Value{types.ObjectValueMust(
			map[string]attr.Type{"a": types.NumberType},
			map[string]attr.Value{"a": types.NumberValue(big.NewFloat(1))},
		)},
	)
	got, err := runNDJSONEncode(t, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "{\"a\":1}\n" {
		t.Errorf("want %q, got %q", "{\"a\":1}\n", got)
	}
}

func TestNDJSONEncode_EmptyList(t *testing.T) {
	in := types.TupleValueMust([]attr.Type{}, []attr.Value{})
	got, err := runNDJSONEncode(t, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("want empty string, got %q", got)
	}
}

func TestNDJSONEncode_RejectsScalar(t *testing.T) {
	_, err := runNDJSONEncode(t, types.StringValue("not a list"))
	if err == nil {
		t.Error("expected error when encoding non-list input")
	}
}
