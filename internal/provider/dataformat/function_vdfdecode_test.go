package dataformat

import (
	"context"
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
