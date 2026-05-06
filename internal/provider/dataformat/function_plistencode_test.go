package dataformat

import (
	"context"
	"encoding/base64"
	"math/big"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runPlistEncode(t *testing.T, value attr.Value, formatStr ...string) (string, *function.FuncError) {
	t.Helper()
	f := &PlistEncodeFunction{}

	var opts []attr.Value
	if len(formatStr) == 1 {
		opts = append(opts, makeFormatOpts(formatStr[0]))
	} else if len(formatStr) > 1 {
		// Pass two options objects to trigger the "too many" error.
		opts = append(opts, makeFormatOpts(formatStr[0]), makeFormatOpts(formatStr[1]))
	}

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

func makeFormatOpts(format string) attr.Value {
	return types.ObjectValueMust(
		map[string]attr.Type{"format": types.StringType},
		map[string]attr.Value{"format": types.StringValue(format)},
	)
}

func TestPlistEncode_SimpleDict(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"Name":    types.StringType,
			"Version": types.NumberType,
			"Enabled": types.BoolType,
		},
		map[string]attr.Value{
			"Name":    types.StringValue("Test"),
			"Version": types.NumberValue(big.NewFloat(1)),
			"Enabled": types.BoolValue(true),
		},
	)

	result, err := runPlistEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "<string>Test</string>") {
		t.Errorf("expected string element in output:\n%s", result)
	}
	if !strings.Contains(result, "<integer>1</integer>") {
		t.Errorf("expected integer element in output:\n%s", result)
	}
	if !strings.Contains(result, "<true/>") {
		t.Errorf("expected true element in output:\n%s", result)
	}
}

func TestPlistEncode_Float(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"Rating": types.NumberType,
		},
		map[string]attr.Value{
			"Rating": types.NumberValue(big.NewFloat(4.5)),
		},
	)

	result, err := runPlistEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "<real>4.5</real>") {
		t.Errorf("expected real element in output:\n%s", result)
	}
}

func TestPlistEncode_WithTaggedDate(t *testing.T) {
	dateObj := types.ObjectValueMust(
		map[string]attr.Type{
			plistTypeKey:  types.StringType,
			plistValueKey: types.StringType,
		},
		map[string]attr.Value{
			plistTypeKey:  types.StringValue(plistTypeDate),
			plistValueKey: types.StringValue("2025-06-01T00:00:00Z"),
		},
	)

	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"ExpirationDate": dateObj.Type(nil),
		},
		map[string]attr.Value{
			"ExpirationDate": dateObj,
		},
	)

	result, err := runPlistEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "<date>") {
		t.Errorf("expected date element in output:\n%s", result)
	}
}

func TestPlistEncode_WithTaggedData(t *testing.T) {
	dataObj := types.ObjectValueMust(
		map[string]attr.Type{
			plistTypeKey:  types.StringType,
			plistValueKey: types.StringType,
		},
		map[string]attr.Value{
			plistTypeKey:  types.StringValue(plistTypeData),
			plistValueKey: types.StringValue("SGVsbG8gV29ybGQ="),
		},
	)

	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"Certificate": dataObj.Type(nil),
		},
		map[string]attr.Value{
			"Certificate": dataObj,
		},
	)

	result, err := runPlistEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "<data>") {
		t.Errorf("expected data element in output:\n%s", result)
	}
}

func TestPlistEncode_InvalidFormat(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{},
		map[string]attr.Value{},
	)
	_, err := runPlistEncode(t, obj, "yaml")
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
}

func TestPlistEncode_OpenStep(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"Name": types.StringType,
		},
		map[string]attr.Value{
			"Name": types.StringValue("Test"),
		},
	)
	result, err := runPlistEncode(t, obj, "openstep")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Name") || !strings.Contains(result, "Test") {
		t.Errorf("expected Name=Test in openstep output:\n%s", result)
	}
}

func TestPlistEncode_GNUStepAlias(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"Key": types.StringType,
		},
		map[string]attr.Value{
			"Key": types.StringValue("Val"),
		},
	)
	_, err := runPlistEncode(t, obj, "gnustep")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlistEncode_Binary(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"Name": types.StringType,
		},
		map[string]attr.Value{
			"Name": types.StringValue("BinaryTest"),
		},
	)
	result, err := runPlistEncode(t, obj, "binary")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Binary output should be base64-encoded.
	_, decErr := base64.StdEncoding.DecodeString(result)
	if decErr != nil {
		t.Fatalf("expected valid base64 for binary format, got error: %v", decErr)
	}
}

func TestPlistEncode_BinaryRoundTrip(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"Name":    types.StringType,
			"Version": types.NumberType,
			"Enabled": types.BoolType,
		},
		map[string]attr.Value{
			"Name":    types.StringValue("RoundTrip"),
			"Version": types.NumberValue(big.NewFloat(3)),
			"Enabled": types.BoolValue(true),
		},
	)

	// Encode as binary.
	b64, encErr := runPlistEncode(t, obj, "binary")
	if encErr != nil {
		t.Fatalf("encode error: %v", encErr)
	}

	// Decode the base64 binary plist.
	decoded, decErr := runPlistDecode(t, b64)
	if decErr != nil {
		t.Fatalf("decode error: %v", decErr)
	}

	decodedObj := decoded.UnderlyingValue().(types.Object)
	attrs := decodedObj.Attributes()

	name := attrs["Name"].(types.String).ValueString()
	if name != "RoundTrip" {
		t.Errorf("expected Name='RoundTrip', got %q", name)
	}
}

func TestPlistEncode_WithTaggedReal(t *testing.T) {
	realObj := types.ObjectValueMust(
		map[string]attr.Type{
			plistTypeKey:  types.StringType,
			plistValueKey: types.StringType,
		},
		map[string]attr.Value{
			plistTypeKey:  types.StringValue(plistTypeReal),
			plistValueKey: types.StringValue("2"),
		},
	)

	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"Scale": realObj.Type(nil),
		},
		map[string]attr.Value{
			"Scale": realObj,
		},
	)

	result, err := runPlistEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "<real>2</real>") {
		t.Errorf("expected <real>2</real> in output:\n%s", result)
	}
}

func TestPlistEncode_RoundTrip(t *testing.T) {
	input := testPlistXML

	decoded, decErr := runPlistDecode(t, input)
	if decErr != nil {
		t.Fatalf("decode error: %v", decErr)
	}

	encoded, encErr := runPlistEncode(t, decoded.UnderlyingValue())
	if encErr != nil {
		t.Fatalf("encode error: %v", encErr)
	}

	if !strings.Contains(encoded, "<string>Test Profile</string>") {
		t.Errorf("expected 'Test Profile' in round-trip output:\n%s", encoded)
	}
	if !strings.Contains(encoded, "<integer>1</integer>") {
		t.Errorf("expected integer 1 in round-trip output:\n%s", encoded)
	}
	if !strings.Contains(encoded, "<true/>") {
		t.Errorf("expected true in round-trip output:\n%s", encoded)
	}
	// 4.5 is fractional, should stay as <real>
	if !strings.Contains(encoded, "<real>4.5</real>") {
		t.Errorf("expected <real>4.5</real> in round-trip output:\n%s", encoded)
	}
}

func runPlistEncodeWithOpts(t *testing.T, value attr.Value, opts ...attr.Value) (string, *function.FuncError) {
	t.Helper()
	f := &PlistEncodeFunction{}

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

func TestPlistEncode_WithComments(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"Name":    types.StringType,
			"Version": types.NumberType,
		},
		map[string]attr.Value{
			"Name":    types.StringValue("Test"),
			"Version": types.NumberValue(big.NewFloat(1)),
		},
	)

	comments := types.ObjectValueMust(
		map[string]attr.Type{
			"Name":    types.StringType,
			"Version": types.StringType,
		},
		map[string]attr.Value{
			"Name":    types.StringValue("Application name"),
			"Version": types.StringValue("Current version number"),
		},
	)

	opts := types.ObjectValueMust(
		map[string]attr.Type{"comments": comments.Type(nil)},
		map[string]attr.Value{"comments": comments},
	)

	result, err := runPlistEncodeWithOpts(t, obj, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "<!-- Application name -->") {
		t.Errorf("expected comment for Name:\n%s", result)
	}
	if !strings.Contains(result, "<!-- Current version number -->") {
		t.Errorf("expected comment for Version:\n%s", result)
	}
	// Comments should appear before their <key> lines.
	commentIdx := strings.Index(result, "<!-- Application name -->")
	keyIdx := strings.Index(result, "<key>Name</key>")
	if commentIdx > keyIdx {
		t.Errorf("comment should appear before <key>:\n%s", result)
	}
}

func TestPlistEncode_CommentsOnlyXML(t *testing.T) {
	// Comments should be silently ignored for non-XML formats.
	obj := types.ObjectValueMust(
		map[string]attr.Type{"Key": types.StringType},
		map[string]attr.Value{"Key": types.StringValue("Val")},
	)

	comments := types.ObjectValueMust(
		map[string]attr.Type{"Key": types.StringType},
		map[string]attr.Value{"Key": types.StringValue("A comment")},
	)

	opts := types.ObjectValueMust(
		map[string]attr.Type{
			"format":   types.StringType,
			"comments": comments.Type(nil),
		},
		map[string]attr.Value{
			"format":   types.StringValue("openstep"),
			"comments": comments,
		},
	)

	result, err := runPlistEncodeWithOpts(t, obj, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// OpenStep format should not have XML comments.
	if strings.Contains(result, "<!--") {
		t.Errorf("expected no XML comments in openstep format:\n%s", result)
	}
}

func TestPlistEncode_CommentEscapesDoubleHyphen(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{"Key": types.StringType},
		map[string]attr.Value{"Key": types.StringValue("Val")},
	)

	// Comment containing "--" which is illegal inside XML comments.
	comments := types.ObjectValueMust(
		map[string]attr.Type{"Key": types.StringType},
		map[string]attr.Value{"Key": types.StringValue("This -- breaks XML")},
	)

	opts := types.ObjectValueMust(
		map[string]attr.Type{"comments": comments.Type(nil)},
		map[string]attr.Value{"comments": comments},
	)

	result, err := runPlistEncodeWithOpts(t, obj, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The "--" should be escaped.
	if strings.Contains(result, "<!-- This -- breaks XML -->") {
		t.Errorf("expected -- to be escaped in XML comment:\n%s", result)
	}
	if !strings.Contains(result, "<!--") {
		t.Errorf("expected comment to still be present:\n%s", result)
	}
}

func TestPlistEncode_TooManyFormatArgs(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{},
		map[string]attr.Value{},
	)
	_, err := runPlistEncode(t, obj, "xml", "binary")
	if err == nil {
		t.Fatal("expected error for too many format args")
	}
}

func TestPlistEncode_OpenStepRoundTrip(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"Name":    types.StringType,
			"Version": types.NumberType,
			"Enabled": types.BoolType,
		},
		map[string]attr.Value{
			"Name":    types.StringValue("OpenStep RT"),
			"Version": types.NumberValue(big.NewFloat(7)),
			"Enabled": types.BoolValue(true),
		},
	)

	encoded, encErr := runPlistEncode(t, obj, "openstep")
	if encErr != nil {
		t.Fatalf("encode error: %v", encErr)
	}

	decoded, decErr := runPlistDecode(t, encoded)
	if decErr != nil {
		t.Fatalf("decode error: %v", decErr)
	}

	decodedObj := decoded.UnderlyingValue().(types.Object)
	attrs := decodedObj.Attributes()

	name := attrs["Name"].(types.String).ValueString()
	if name != "OpenStep RT" {
		t.Errorf("expected Name='OpenStep RT', got %q", name)
	}
}
