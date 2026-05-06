package dataformat

import (
	"context"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runPlistDate(t *testing.T, input string) (types.Dynamic, *function.FuncError) {
	t.Helper()
	f := &PlistDateFunction{}

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

func TestPlistDate_Valid(t *testing.T) {
	result, err := runPlistDate(t, "2025-06-01T00:00:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj, ok := result.UnderlyingValue().(types.Object)
	if !ok {
		t.Fatalf("expected Object, got %T", result.UnderlyingValue())
	}

	attrs := obj.Attributes()
	typeVal := attrs[plistTypeKey].(types.String).ValueString()
	if typeVal != plistTypeDate {
		t.Errorf("expected __plist_type=%q, got %q", plistTypeDate, typeVal)
	}
	valueVal := attrs[plistValueKey].(types.String).ValueString()
	if valueVal != "2025-06-01T00:00:00Z" {
		t.Errorf("expected value=%q, got %q", "2025-06-01T00:00:00Z", valueVal)
	}
}

func TestPlistDate_WithTimezone(t *testing.T) {
	_, err := runPlistDate(t, "2025-06-01T12:30:00-07:00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlistDate_Invalid(t *testing.T) {
	_, err := runPlistDate(t, "not a date")
	if err == nil {
		t.Fatal("expected error for invalid date")
	}
}

func TestPlistDate_InvalidFormat(t *testing.T) {
	_, err := runPlistDate(t, "2025-06-01")
	if err == nil {
		t.Fatal("expected error for date without time")
	}
}

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

func runPlistReal(t *testing.T, value *big.Float) (types.Dynamic, *function.FuncError) {
	t.Helper()
	f := &PlistRealFunction{}

	args := function.NewArgumentsData([]attr.Value{types.NumberValue(value)})
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

func TestPlistReal_WholeNumber(t *testing.T) {
	result, err := runPlistReal(t, big.NewFloat(2))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj, ok := result.UnderlyingValue().(types.Object)
	if !ok {
		t.Fatalf("expected Object, got %T", result.UnderlyingValue())
	}

	attrs := obj.Attributes()
	typeVal := attrs[plistTypeKey].(types.String).ValueString()
	if typeVal != plistTypeReal {
		t.Errorf("expected __plist_type=%q, got %q", plistTypeReal, typeVal)
	}
	valueVal := attrs[plistValueKey].(types.String).ValueString()
	if valueVal != "2" {
		t.Errorf("expected value=%q, got %q", "2", valueVal)
	}
}

func TestPlistReal_Fractional(t *testing.T) {
	result, err := runPlistReal(t, big.NewFloat(3.14))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj, ok := result.UnderlyingValue().(types.Object)
	if !ok {
		t.Fatalf("expected Object, got %T", result.UnderlyingValue())
	}

	attrs := obj.Attributes()
	typeVal := attrs[plistTypeKey].(types.String).ValueString()
	if typeVal != plistTypeReal {
		t.Errorf("expected __plist_type=%q, got %q", plistTypeReal, typeVal)
	}
}

func TestPlistReal_Zero(t *testing.T) {
	_, err := runPlistReal(t, big.NewFloat(0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
