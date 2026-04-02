package provider

import (
	"context"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
