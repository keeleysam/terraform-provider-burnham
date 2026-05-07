package dataformat

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runHCLDecode(t *testing.T, input string) (types.Dynamic, *function.FuncError) {
	t.Helper()
	f := &HCLDecodeFunction{}
	args := function.NewArgumentsData([]attr.Value{types.StringValue(input)})
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.DynamicNull())}
	f.Run(context.Background(), req, resp)
	if resp.Error != nil {
		return types.DynamicNull(), resp.Error
	}
	return resp.Result.Value().(types.Dynamic), nil
}

func runHCLEncode(t *testing.T, value attr.Value) (string, *function.FuncError) {
	t.Helper()
	f := &HCLEncodeFunction{}
	args := function.NewArgumentsData([]attr.Value{types.DynamicValue(value)})
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.StringValue(""))}
	f.Run(context.Background(), req, resp)
	if resp.Error != nil {
		return "", resp.Error
	}
	return resp.Result.Value().(types.String).ValueString(), nil
}

func TestHCLDecode_RejectsBlockSyntax(t *testing.T) {
	_, err := runHCLDecode(t, "name = \"web\"\nprovisioner \"local\" { x = 1 }\n")
	if err == nil {
		t.Fatal("expected error when blocks are present")
	}
	if !strings.Contains(err.Error(), "block") {
		t.Errorf("error should mention blocks, got: %s", err.Error())
	}
}

func TestHCLDecode_Empty(t *testing.T) {
	got, err := runHCLDecode(t, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	obj := got.UnderlyingValue().(types.Object)
	if len(obj.Attributes()) != 0 {
		t.Errorf("want empty object, got %v", obj.Attributes())
	}
}

func TestHCLEncode_RejectsInvalidIdentifier(t *testing.T) {
	// HCL identifiers allow hyphens, so "foo-bar" is actually valid. Whitespace and dots are not — use those for the negative test.
	in := types.ObjectValueMust(
		map[string]attr.Type{"foo bar": types.StringType},
		map[string]attr.Value{"foo bar": types.StringValue("v")},
	)
	_, err := runHCLEncode(t, in)
	if err == nil {
		t.Error("expected error for non-identifier attribute name")
	}
}

func TestHCLEncode_NestedObject(t *testing.T) {
	in := types.ObjectValueMust(
		map[string]attr.Type{
			"server": types.ObjectType{AttrTypes: map[string]attr.Type{
				"host": types.StringType,
				"port": types.NumberType,
			}},
		},
		map[string]attr.Value{
			"server": types.ObjectValueMust(
				map[string]attr.Type{"host": types.StringType, "port": types.NumberType},
				map[string]attr.Value{"host": types.StringValue("localhost"), "port": types.NumberValue(big.NewFloat(8080))},
			),
		},
	)
	got, err := runHCLEncode(t, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "server") || !strings.Contains(got, "host") || !strings.Contains(got, "localhost") {
		t.Errorf("nested object missing from output: %q", got)
	}
}
