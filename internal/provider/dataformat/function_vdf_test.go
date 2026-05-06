package dataformat

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runVDFDecode(t *testing.T, input string) (types.Dynamic, *function.FuncError) {
	t.Helper()
	f := &VDFDecodeFunction{}

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

func TestVDFDecode_Basic(t *testing.T) {
	input := "\"AppState\"\n{\n\t\"appid\"\t\t\"730\"\n\t\"name\"\t\t\"Counter-Strike 2\"\n}\n"

	result, err := runVDFDecode(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj := result.UnderlyingValue().(types.Object)
	app := obj.Attributes()["AppState"].(types.Object)

	appid := app.Attributes()["appid"].(types.String).ValueString()
	if appid != "730" {
		t.Errorf("expected appid='730', got %q", appid)
	}

	name := app.Attributes()["name"].(types.String).ValueString()
	if name != "Counter-Strike 2" {
		t.Errorf("expected name='Counter-Strike 2', got %q", name)
	}
}

func TestVDFDecode_Nested(t *testing.T) {
	input := "\"Root\"\n{\n\t\"Child\"\n\t{\n\t\t\"key\"\t\t\"value\"\n\t}\n}\n"

	result, err := runVDFDecode(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj := result.UnderlyingValue().(types.Object)
	root := obj.Attributes()["Root"].(types.Object)
	child := root.Attributes()["Child"].(types.Object)
	val := child.Attributes()["key"].(types.String).ValueString()
	if val != "value" {
		t.Errorf("expected key='value', got %q", val)
	}
}

func TestVDFDecode_WithComments(t *testing.T) {
	input := "\"Config\"\n{\n\t// This is a comment\n\t\"key\"\t\t\"value\"\n}\n"

	result, err := runVDFDecode(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj := result.UnderlyingValue().(types.Object)
	config := obj.Attributes()["Config"].(types.Object)
	val := config.Attributes()["key"].(types.String).ValueString()
	if val != "value" {
		t.Errorf("expected key='value', got %q", val)
	}
}

func TestVDFDecode_Invalid(t *testing.T) {
	_, err := runVDFDecode(t, "this is not vdf")
	if err == nil {
		t.Fatal("expected error for invalid input")
	}
}

func TestVDFDecode_SteamLibrary(t *testing.T) {
	// Real-world Steam libraryfolders.vdf snippet.
	input := `"libraryfolders"
{
	"0"
	{
		"path"		"/Applications/Steam"
		"label"		""
		"apps"
		{
			"730"		"26685592507"
			"440"		"21899556124"
		}
	}
}`

	result, err := runVDFDecode(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj := result.UnderlyingValue().(types.Object)
	lib := obj.Attributes()["libraryfolders"].(types.Object)
	folder := lib.Attributes()["0"].(types.Object)
	path := folder.Attributes()["path"].(types.String).ValueString()
	if path != "/Applications/Steam" {
		t.Errorf("expected path='/Applications/Steam', got %q", path)
	}
}

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
