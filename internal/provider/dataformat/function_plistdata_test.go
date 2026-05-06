package dataformat

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runPlistData(t *testing.T, input string) (types.Dynamic, *function.FuncError) {
	t.Helper()
	f := &PlistDataFunction{}

	args := function.NewArgumentsData([]attr.Value{types.StringValue(input)})
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.DynamicNull())}

	f.Run(context.Background(), req, resp)

	if resp.Error != nil {
		return types.DynamicNull(), resp.Error
	}

	result, ok := resp.Result.Value().(types.Dynamic)
	if !ok {
		t.Fatalf("expected Dynamic result, got %T", resp.Result.Value())
	}
	return result, nil
}

func TestPlistData_Valid(t *testing.T) {
	result, err := runPlistData(t, "SGVsbG8gV29ybGQ=")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj, ok := result.UnderlyingValue().(types.Object)
	if !ok {
		t.Fatalf("expected Object, got %T", result.UnderlyingValue())
	}

	attrs := obj.Attributes()
	typeVal := attrs[plistTypeKey].(types.String).ValueString()
	if typeVal != plistTypeData {
		t.Errorf("expected __plist_type=%q, got %q", plistTypeData, typeVal)
	}
	valueVal := attrs[plistValueKey].(types.String).ValueString()
	if valueVal != "SGVsbG8gV29ybGQ=" {
		t.Errorf("expected value=%q, got %q", "SGVsbG8gV29ybGQ=", valueVal)
	}
}

func TestPlistData_InvalidBase64(t *testing.T) {
	_, err := runPlistData(t, "not valid base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestPlistData_Empty(t *testing.T) {
	// Empty string is valid base64 (decodes to empty bytes).
	_, err := runPlistData(t, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
