package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runINIEncode(t *testing.T, value attr.Value) (string, *function.FuncError) {
	t.Helper()
	f := &INIEncodeFunction{}

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

func TestINIEncode_BasicSections(t *testing.T) {
	dbSection := types.ObjectValueMust(
		map[string]attr.Type{"host": types.StringType, "port": types.StringType},
		map[string]attr.Value{"host": types.StringValue("localhost"), "port": types.StringValue("5432")},
	)

	obj := types.ObjectValueMust(
		map[string]attr.Type{"database": dbSection.Type(nil)},
		map[string]attr.Value{"database": dbSection},
	)

	result, err := runINIEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "[database]") {
		t.Errorf("expected [database] section in output:\n%s", result)
	}
	if !strings.Contains(result, "host = localhost") {
		t.Errorf("expected host = localhost in output:\n%s", result)
	}
	if !strings.Contains(result, "port = 5432") {
		t.Errorf("expected port = 5432 in output:\n%s", result)
	}
}

func TestINIEncode_GlobalKeys(t *testing.T) {
	globalSection := types.ObjectValueMust(
		map[string]attr.Type{"key": types.StringType},
		map[string]attr.Value{"key": types.StringValue("global_value")},
	)
	namedSection := types.ObjectValueMust(
		map[string]attr.Type{"other": types.StringType},
		map[string]attr.Value{"other": types.StringValue("thing")},
	)

	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"":        globalSection.Type(nil),
			"section": namedSection.Type(nil),
		},
		map[string]attr.Value{
			"":        globalSection,
			"section": namedSection,
		},
	)

	result, err := runINIEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "key = global_value") {
		t.Errorf("expected global key in output:\n%s", result)
	}
	if !strings.Contains(result, "[section]") {
		t.Errorf("expected [section] in output:\n%s", result)
	}

	// Global keys should come before section headers.
	globalIdx := strings.Index(result, "key = global_value")
	sectionIdx := strings.Index(result, "[section]")
	if globalIdx > sectionIdx {
		t.Errorf("expected global keys before section header:\n%s", result)
	}
}

func TestINIEncode_RoundTrip(t *testing.T) {
	input := `[database]
host = localhost
port = 5432
`

	decoded, decErr := runINIDecode(t, input)
	if decErr != nil {
		t.Fatalf("decode error: %v", decErr)
	}

	encoded, encErr := runINIEncode(t, decoded.UnderlyingValue())
	if encErr != nil {
		t.Fatalf("encode error: %v", encErr)
	}

	if !strings.Contains(encoded, "[database]") {
		t.Errorf("expected [database] in round-trip output:\n%s", encoded)
	}
	if !strings.Contains(encoded, "host = localhost") {
		t.Errorf("expected host = localhost in round-trip output:\n%s", encoded)
	}
}

func TestINIEncode_NotAnObject(t *testing.T) {
	_, err := runINIEncode(t, types.StringValue("not an object"))
	if err == nil {
		t.Fatal("expected error for non-object input")
	}
}
