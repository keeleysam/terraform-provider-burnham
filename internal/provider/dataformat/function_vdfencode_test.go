package dataformat

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runVDFEncode(t *testing.T, value attr.Value) (string, *function.FuncError) {
	t.Helper()
	f := &VDFEncodeFunction{}

	args := function.NewArgumentsData([]attr.Value{types.DynamicValue(value)})
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

func TestVDFEncode_Basic(t *testing.T) {
	inner := types.ObjectValueMust(
		map[string]attr.Type{"appid": types.StringType, "name": types.StringType},
		map[string]attr.Value{"appid": types.StringValue("730"), "name": types.StringValue("Counter-Strike 2")},
	)
	obj := types.ObjectValueMust(
		map[string]attr.Type{"AppState": inner.Type(nil)},
		map[string]attr.Value{"AppState": inner},
	)

	result, err := runVDFEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, `"AppState"`) {
		t.Errorf("expected AppState key:\n%s", result)
	}
	if !strings.Contains(result, `"appid"`) && !strings.Contains(result, `"730"`) {
		t.Errorf("expected appid value:\n%s", result)
	}
	if !strings.Contains(result, "{") || !strings.Contains(result, "}") {
		t.Errorf("expected braces:\n%s", result)
	}
}

func TestVDFEncode_Nested(t *testing.T) {
	child := types.ObjectValueMust(
		map[string]attr.Type{"key": types.StringType},
		map[string]attr.Value{"key": types.StringValue("value")},
	)
	root := types.ObjectValueMust(
		map[string]attr.Type{"Child": child.Type(nil)},
		map[string]attr.Value{"Child": child},
	)
	obj := types.ObjectValueMust(
		map[string]attr.Type{"Root": root.Type(nil)},
		map[string]attr.Value{"Root": root},
	)

	result, err := runVDFEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, `"Root"`) {
		t.Errorf("expected Root:\n%s", result)
	}
	if !strings.Contains(result, `"Child"`) {
		t.Errorf("expected Child:\n%s", result)
	}
	if !strings.Contains(result, `"key"`) {
		t.Errorf("expected key:\n%s", result)
	}
}

func TestVDFEncode_RoundTrip(t *testing.T) {
	input := "\"Config\"\n{\n\t\"key\"\t\t\"value\"\n\t\"number\"\t\t\"42\"\n}\n"

	decoded, decErr := runVDFDecode(t, input)
	if decErr != nil {
		t.Fatalf("decode error: %v", decErr)
	}

	encoded, encErr := runVDFEncode(t, decoded.UnderlyingValue())
	if encErr != nil {
		t.Fatalf("encode error: %v", encErr)
	}

	if !strings.Contains(encoded, `"key"`) || !strings.Contains(encoded, `"value"`) {
		t.Errorf("expected key/value in round-trip:\n%s", encoded)
	}
}

func TestVDFEncode_NotAnObject(t *testing.T) {
	_, err := runVDFEncode(t, types.StringValue("not an object"))
	if err == nil {
		t.Fatal("expected error for non-object input")
	}
}
