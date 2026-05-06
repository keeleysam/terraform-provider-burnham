package dataformat

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runRegDecode(t *testing.T, input string) (types.Dynamic, *function.FuncError) {
	t.Helper()
	f := &RegDecodeFunction{}

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

func getRegKeyAttrs(t *testing.T, result types.Dynamic, keyPath string) map[string]attr.Value {
	t.Helper()
	obj := result.UnderlyingValue().(types.Object)
	keyObj, ok := obj.Attributes()[keyPath].(types.Object)
	if !ok {
		t.Fatalf("key %q not found or not an object", keyPath)
	}
	return keyObj.Attributes()
}

func assertRegTaggedType(t *testing.T, val attr.Value, expectedType string) {
	t.Helper()
	obj, ok := val.(types.Object)
	if !ok {
		t.Fatalf("expected tagged object, got %T", val)
	}
	typeVal := obj.Attributes()[regTypeKey].(types.String).ValueString()
	if typeVal != expectedType {
		t.Errorf("expected __reg_type=%q, got %q", expectedType, typeVal)
	}
}

func TestRegDecode_BasicV5(t *testing.T) {
	input := "Windows Registry Editor Version 5.00\r\n\r\n[HKEY_LOCAL_MACHINE\\SOFTWARE\\Test]\r\n\"Name\"=\"Hello\"\r\n\"Count\"=dword:0000002a\r\n"

	result, err := runRegDecode(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	attrs := getRegKeyAttrs(t, result, "HKEY_LOCAL_MACHINE\\SOFTWARE\\Test")

	name := attrs["Name"].(types.String).ValueString()
	if name != "Hello" {
		t.Errorf("expected Name='Hello', got %q", name)
	}

	assertRegTaggedType(t, attrs["Count"], regTypeDword)
	countVal := attrs["Count"].(types.Object).Attributes()[regValueKey].(types.String).ValueString()
	if countVal != "42" {
		t.Errorf("expected dword value='42', got %q", countVal)
	}
}

func TestRegDecode_V4(t *testing.T) {
	input := "REGEDIT4\r\n\r\n[HKEY_LOCAL_MACHINE\\SOFTWARE\\Test]\r\n\"Key\"=\"Value\"\r\n"

	result, err := runRegDecode(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj := result.UnderlyingValue().(types.Object)
	if _, ok := obj.Attributes()["HKEY_LOCAL_MACHINE\\SOFTWARE\\Test"]; !ok {
		t.Error("expected key path in result")
	}
}

func TestRegDecode_DefaultValue(t *testing.T) {
	input := "Windows Registry Editor Version 5.00\r\n\r\n[HKEY_LOCAL_MACHINE\\SOFTWARE\\Test]\r\n@=\"Default\"\r\n"

	result, err := runRegDecode(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	attrs := getRegKeyAttrs(t, result, "HKEY_LOCAL_MACHINE\\SOFTWARE\\Test")
	defVal := attrs["@"].(types.String).ValueString()
	if defVal != "Default" {
		t.Errorf("expected @='Default', got %q", defVal)
	}
}

func TestRegDecode_AllTypes(t *testing.T) {
	input := "Windows Registry Editor Version 5.00\r\n\r\n" +
		"[HKEY_LOCAL_MACHINE\\SOFTWARE\\AllTypes]\r\n" +
		"\"String\"=\"hello\"\r\n" +
		"\"Dword\"=dword:0000002a\r\n" +
		"\"Qword\"=hex(b):2a,00,00,00,00,00,00,00\r\n" +
		"\"Binary\"=hex:48,65,6c,6c,6f\r\n" +
		"\"Multi\"=hex(7):68,00,65,00,6c,00,6c,00,6f,00,00,00,77,00,6f,00,72,00,6c,00,64,00,00,00,00,00\r\n" +
		"\"Expand\"=hex(2):25,00,53,00,79,00,73,00,74,00,65,00,6d,00,52,00,6f,00,6f,00,74,00,25,00,00,00\r\n"

	result, err := runRegDecode(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	attrs := getRegKeyAttrs(t, result, "HKEY_LOCAL_MACHINE\\SOFTWARE\\AllTypes")

	// REG_SZ
	if attrs["String"].(types.String).ValueString() != "hello" {
		t.Error("REG_SZ not decoded correctly")
	}

	// REG_DWORD
	assertRegTaggedType(t, attrs["Dword"], regTypeDword)

	// REG_QWORD
	assertRegTaggedType(t, attrs["Qword"], regTypeQword)

	// REG_BINARY
	assertRegTaggedType(t, attrs["Binary"], regTypeBinary)
	binVal := attrs["Binary"].(types.Object).Attributes()[regValueKey].(types.String).ValueString()
	if binVal != "48656c6c6f" {
		t.Errorf("expected binary hex '48656c6c6f', got %q", binVal)
	}

	// REG_MULTI_SZ
	assertRegTaggedType(t, attrs["Multi"], regTypeMultiSz)

	// REG_EXPAND_SZ
	assertRegTaggedType(t, attrs["Expand"], regTypeExpandSz)
	expandVal := attrs["Expand"].(types.Object).Attributes()[regValueKey].(types.String).ValueString()
	if expandVal != "%SystemRoot%" {
		t.Errorf("expected expand_sz value '%%SystemRoot%%', got %q", expandVal)
	}
}

func TestRegDecode_MultipleKeys(t *testing.T) {
	input := "Windows Registry Editor Version 5.00\r\n\r\n" +
		"[HKEY_LOCAL_MACHINE\\SOFTWARE\\App1]\r\n\"Name\"=\"First\"\r\n\r\n" +
		"[HKEY_LOCAL_MACHINE\\SOFTWARE\\App2]\r\n\"Name\"=\"Second\"\r\n"

	result, err := runRegDecode(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj := result.UnderlyingValue().(types.Object)
	if len(obj.Attributes()) != 2 {
		t.Errorf("expected 2 keys, got %d", len(obj.Attributes()))
	}
}

func TestRegDecode_RoundTrip(t *testing.T) {
	input := "Windows Registry Editor Version 5.00\r\n\r\n" +
		"[HKEY_LOCAL_MACHINE\\SOFTWARE\\Test]\r\n" +
		"\"Name\"=\"Hello\"\r\n" +
		"\"Count\"=dword:0000002a\r\n"

	decoded, decErr := runRegDecode(t, input)
	if decErr != nil {
		t.Fatalf("decode error: %v", decErr)
	}

	encoded, encErr := runRegEncode(t, decoded.UnderlyingValue())
	if encErr != nil {
		t.Fatalf("encode error: %v", encErr)
	}

	if !strings.Contains(encoded, `"Name"="Hello"`) {
		t.Errorf("expected Name=Hello in round-trip:\n%s", encoded)
	}
	if !strings.Contains(encoded, "dword:0000002a") {
		t.Errorf("expected dword:0000002a in round-trip:\n%s", encoded)
	}
}

func TestRegDecode_Invalid(t *testing.T) {
	_, err := runRegDecode(t, "this is not a reg file")
	if err == nil {
		t.Fatal("expected error for invalid input")
	}
}

func TestRegDecode_EmptyKeys(t *testing.T) {
	input := "Windows Registry Editor Version 5.00\r\n\r\n[HKEY_LOCAL_MACHINE\\SOFTWARE\\Empty]\r\n"

	result, err := runRegDecode(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj := result.UnderlyingValue().(types.Object)
	if len(obj.Attributes()) != 0 {
		t.Errorf("expected no keys for empty registry key, got %d", len(obj.Attributes()))
	}
}
