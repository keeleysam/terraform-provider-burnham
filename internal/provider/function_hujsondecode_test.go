package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runHuJSONDecode(t *testing.T, input string) (types.Dynamic, *function.FuncError) {
	t.Helper()
	f := &HuJSONDecodeFunction{}

	// No variadic, so just the one parameter.
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

func TestHuJSONDecode_Simple(t *testing.T) {
	input := `{
		// This is a comment
		"name": "test",
		"count": 42,
	}`

	result, err := runHuJSONDecode(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsNull() {
		t.Fatal("expected non-null result")
	}
}

func TestHuJSONDecode_TrailingCommas(t *testing.T) {
	input := `{"items": [1, 2, 3,], "enabled": true,}`

	_, err := runHuJSONDecode(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHuJSONDecode_BlockComments(t *testing.T) {
	input := `{
		/* block comment */
		"key": "value"
	}`

	_, err := runHuJSONDecode(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHuJSONDecode_InvalidInput(t *testing.T) {
	_, err := runHuJSONDecode(t, `{not valid at all`)
	if err == nil {
		t.Fatal("expected error for invalid input")
	}
}

func TestHuJSONDecode_StandardJSON(t *testing.T) {
	input := `{"a": 1, "b": [true, false]}`

	_, err := runHuJSONDecode(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHuJSONDecode_EmptyString(t *testing.T) {
	_, err := runHuJSONDecode(t, "")
	if err == nil {
		t.Fatal("expected error for empty string input")
	}
}
