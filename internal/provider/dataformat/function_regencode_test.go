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

func runRegEncode(t *testing.T, value attr.Value) (string, *function.FuncError) {
	t.Helper()
	return runRegEncodeWithOpts(t, value)
}

func makeRegTagged(regType string, value attr.Value) attr.Value {
	return types.ObjectValueMust(
		map[string]attr.Type{regTypeKey: types.StringType, regValueKey: value.Type(nil)},
		map[string]attr.Value{regTypeKey: types.StringValue(regType), regValueKey: value},
	)
}

func TestRegEncode_BasicString(t *testing.T) {
	valuesObj := types.ObjectValueMust(
		map[string]attr.Type{"Name": types.StringType},
		map[string]attr.Value{"Name": types.StringValue("Hello")},
	)
	obj := types.ObjectValueMust(
		map[string]attr.Type{"HKEY_LOCAL_MACHINE\\SOFTWARE\\Test": valuesObj.Type(nil)},
		map[string]attr.Value{"HKEY_LOCAL_MACHINE\\SOFTWARE\\Test": valuesObj},
	)

	result, err := runRegEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "Windows Registry Editor Version 5.00") {
		t.Errorf("expected Version 5 header:\n%s", result)
	}
	if !strings.Contains(result, `"Name"="Hello"`) {
		t.Errorf("expected string value in output:\n%s", result)
	}
}

func TestRegEncode_WithDword(t *testing.T) {
	dword := makeRegTagged(regTypeDword, types.StringValue("42"))
	valuesObj := types.ObjectValueMust(
		map[string]attr.Type{"Count": dword.Type(nil)},
		map[string]attr.Value{"Count": dword},
	)
	obj := types.ObjectValueMust(
		map[string]attr.Type{"HKEY_LOCAL_MACHINE\\SOFTWARE\\Test": valuesObj.Type(nil)},
		map[string]attr.Value{"HKEY_LOCAL_MACHINE\\SOFTWARE\\Test": valuesObj},
	)

	result, err := runRegEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "dword:0000002a") {
		t.Errorf("expected dword value in output:\n%s", result)
	}
}

func TestRegEncode_WithQword(t *testing.T) {
	qword := makeRegTagged(regTypeQword, types.StringValue("1099511627776"))
	valuesObj := types.ObjectValueMust(
		map[string]attr.Type{"BigNum": qword.Type(nil)},
		map[string]attr.Value{"BigNum": qword},
	)
	obj := types.ObjectValueMust(
		map[string]attr.Type{"HKEY_LOCAL_MACHINE\\SOFTWARE\\Test": valuesObj.Type(nil)},
		map[string]attr.Value{"HKEY_LOCAL_MACHINE\\SOFTWARE\\Test": valuesObj},
	)

	result, err := runRegEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "hex(b):") {
		t.Errorf("expected qword hex(b) in output:\n%s", result)
	}
}

func TestRegEncode_WithBinary(t *testing.T) {
	binary := makeRegTagged(regTypeBinary, types.StringValue("48656c6c6f"))
	valuesObj := types.ObjectValueMust(
		map[string]attr.Type{"Data": binary.Type(nil)},
		map[string]attr.Value{"Data": binary},
	)
	obj := types.ObjectValueMust(
		map[string]attr.Type{"HKEY_LOCAL_MACHINE\\SOFTWARE\\Test": valuesObj.Type(nil)},
		map[string]attr.Value{"HKEY_LOCAL_MACHINE\\SOFTWARE\\Test": valuesObj},
	)

	result, err := runRegEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "hex:48,65,6c,6c,6f") {
		t.Errorf("expected binary hex in output:\n%s", result)
	}
}

func TestRegEncode_WithExpandSz(t *testing.T) {
	expand := makeRegTagged(regTypeExpandSz, types.StringValue("%SystemRoot%\\system32"))
	valuesObj := types.ObjectValueMust(
		map[string]attr.Type{"Path": expand.Type(nil)},
		map[string]attr.Value{"Path": expand},
	)
	obj := types.ObjectValueMust(
		map[string]attr.Type{"HKEY_LOCAL_MACHINE\\SOFTWARE\\Test": valuesObj.Type(nil)},
		map[string]attr.Value{"HKEY_LOCAL_MACHINE\\SOFTWARE\\Test": valuesObj},
	)

	result, err := runRegEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "hex(2):") {
		t.Errorf("expected expand_sz hex(2) in output:\n%s", result)
	}
}

func TestRegEncode_DefaultValue(t *testing.T) {
	valuesObj := types.ObjectValueMust(
		map[string]attr.Type{"@": types.StringType},
		map[string]attr.Value{"@": types.StringValue("Default")},
	)
	obj := types.ObjectValueMust(
		map[string]attr.Type{"HKEY_LOCAL_MACHINE\\SOFTWARE\\Test": valuesObj.Type(nil)},
		map[string]attr.Value{"HKEY_LOCAL_MACHINE\\SOFTWARE\\Test": valuesObj},
	)

	result, err := runRegEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, `@="Default"`) {
		t.Errorf("expected default value in output:\n%s", result)
	}
}

func TestRegEncode_NotAnObject(t *testing.T) {
	_, err := runRegEncode(t, types.StringValue("not an object"))
	if err == nil {
		t.Fatal("expected error for non-object input")
	}
}

// ─── Helper function tests ───────────────────────────────────────

func runRegHelper(t *testing.T, f function.Function, args []attr.Value) (types.Dynamic, *function.FuncError) {
	t.Helper()

	req := function.RunRequest{Arguments: function.NewArgumentsData(args)}
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

func TestRegDword_Helper(t *testing.T) {
	result, err := runRegHelper(t, &RegDwordFunction{}, []attr.Value{types.NumberValue(big.NewFloat(42))})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	obj := result.UnderlyingValue().(types.Object)
	if obj.Attributes()[regTypeKey].(types.String).ValueString() != regTypeDword {
		t.Error("expected __reg_type=dword")
	}
	if obj.Attributes()[regValueKey].(types.String).ValueString() != "42" {
		t.Error("expected value=42")
	}
}

func TestRegQword_Helper(t *testing.T) {
	result, err := runRegHelper(t, &RegQwordFunction{}, []attr.Value{types.NumberValue(big.NewFloat(1099511627776))})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	obj := result.UnderlyingValue().(types.Object)
	if obj.Attributes()[regTypeKey].(types.String).ValueString() != regTypeQword {
		t.Error("expected __reg_type=qword")
	}
}

func TestRegBinary_Helper(t *testing.T) {
	result, err := runRegHelper(t, &RegBinaryFunction{}, []attr.Value{types.StringValue("48656c6c6f")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	obj := result.UnderlyingValue().(types.Object)
	if obj.Attributes()[regTypeKey].(types.String).ValueString() != regTypeBinary {
		t.Error("expected __reg_type=binary")
	}
}

func TestRegBinary_InvalidHex(t *testing.T) {
	_, err := runRegHelper(t, &RegBinaryFunction{}, []attr.Value{types.StringValue("not hex!")})
	if err == nil {
		t.Fatal("expected error for invalid hex")
	}
}

func TestRegMulti_Helper(t *testing.T) {
	list := types.TupleValueMust(
		[]attr.Type{types.StringType, types.StringType},
		[]attr.Value{types.StringValue("hello"), types.StringValue("world")},
	)
	result, err := runRegHelper(t, &RegMultiFunction{}, []attr.Value{types.DynamicValue(list)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	obj := result.UnderlyingValue().(types.Object)
	if obj.Attributes()[regTypeKey].(types.String).ValueString() != regTypeMultiSz {
		t.Error("expected __reg_type=multi_sz")
	}
}

// ─── Comment tests ───────────────────────────────────────────────

func runRegEncodeWithOpts(t *testing.T, value attr.Value, opts ...attr.Value) (string, *function.FuncError) {
	t.Helper()
	f := &RegEncodeFunction{}

	optsElems := make([]attr.Value, len(opts))
	optsTypes := make([]attr.Type, len(opts))
	for i, o := range opts {
		optsElems[i] = types.DynamicValue(o)
		optsTypes[i] = types.DynamicType
	}
	variadicTuple := types.TupleValueMust(optsTypes, optsElems)

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

func TestRegEncode_WithComments(t *testing.T) {
	valuesObj := types.ObjectValueMust(
		map[string]attr.Type{"Name": types.StringType, "Count": types.StringType},
		map[string]attr.Value{"Name": types.StringValue("Hello"), "Count": types.StringValue("42")},
	)
	obj := types.ObjectValueMust(
		map[string]attr.Type{"HKEY_LOCAL_MACHINE\\SOFTWARE\\Test": valuesObj.Type(nil)},
		map[string]attr.Value{"HKEY_LOCAL_MACHINE\\SOFTWARE\\Test": valuesObj},
	)

	valueComments := types.ObjectValueMust(
		map[string]attr.Type{"Name": types.StringType},
		map[string]attr.Value{"Name": types.StringValue("Application name")},
	)
	comments := types.ObjectValueMust(
		map[string]attr.Type{"HKEY_LOCAL_MACHINE\\SOFTWARE\\Test": valueComments.Type(nil)},
		map[string]attr.Value{"HKEY_LOCAL_MACHINE\\SOFTWARE\\Test": valueComments},
	)
	opts := types.ObjectValueMust(
		map[string]attr.Type{"comments": comments.Type(nil)},
		map[string]attr.Value{"comments": comments},
	)

	result, err := runRegEncodeWithOpts(t, obj, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "; Application name") {
		t.Errorf("expected comment in output:\n%s", result)
	}
	// Comment should appear before the value line.
	commentIdx := strings.Index(result, "; Application name")
	nameIdx := strings.Index(result, `"Name"="Hello"`)
	if commentIdx > nameIdx {
		t.Errorf("comment should appear before value:\n%s", result)
	}
}

func TestRegEncode_KeyPathComment(t *testing.T) {
	valuesObj := types.ObjectValueMust(
		map[string]attr.Type{"Key": types.StringType},
		map[string]attr.Value{"Key": types.StringValue("Val")},
	)
	obj := types.ObjectValueMust(
		map[string]attr.Type{"HKEY_LOCAL_MACHINE\\SOFTWARE\\MyApp": valuesObj.Type(nil)},
		map[string]attr.Value{"HKEY_LOCAL_MACHINE\\SOFTWARE\\MyApp": valuesObj},
	)

	comments := types.ObjectValueMust(
		map[string]attr.Type{"HKEY_LOCAL_MACHINE\\SOFTWARE\\MyApp": types.StringType},
		map[string]attr.Value{"HKEY_LOCAL_MACHINE\\SOFTWARE\\MyApp": types.StringValue("Application settings")},
	)
	opts := types.ObjectValueMust(
		map[string]attr.Type{"comments": comments.Type(nil)},
		map[string]attr.Value{"comments": comments},
	)

	result, err := runRegEncodeWithOpts(t, obj, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "; Application settings") {
		t.Errorf("expected key path comment in output:\n%s", result)
	}
}

func TestRegExpandSz_Helper(t *testing.T) {
	result, err := runRegHelper(t, &RegExpandSzFunction{}, []attr.Value{types.StringValue("%SystemRoot%\\system32")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	obj := result.UnderlyingValue().(types.Object)
	if obj.Attributes()[regTypeKey].(types.String).ValueString() != regTypeExpandSz {
		t.Error("expected __reg_type=expand_sz")
	}
	if obj.Attributes()[regValueKey].(types.String).ValueString() != "%SystemRoot%\\system32" {
		t.Error("expected value preserved")
	}
}
