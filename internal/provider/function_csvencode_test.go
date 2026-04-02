package provider

import (
	"context"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runCSVEncode(t *testing.T, rows attr.Value, opts ...attr.Value) (string, *function.FuncError) {
	t.Helper()
	f := &CSVEncodeFunction{}

	optsElems := make([]attr.Value, len(opts))
	optsTypes := make([]attr.Type, len(opts))
	for i, o := range opts {
		optsElems[i] = types.DynamicValue(o)
		optsTypes[i] = types.DynamicType
	}
	variadicTuple := types.TupleValueMust(optsTypes, optsElems)

	args := function.NewArgumentsData([]attr.Value{
		types.DynamicValue(rows),
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

func makeRow(kv map[string]attr.Value) attr.Value {
	attrTypes := make(map[string]attr.Type, len(kv))
	for k, v := range kv {
		attrTypes[k] = v.Type(nil)
	}
	return types.ObjectValueMust(attrTypes, kv)
}

func makeRowList(rows ...attr.Value) attr.Value {
	elemTypes := make([]attr.Type, len(rows))
	for i, r := range rows {
		elemTypes[i] = r.Type(nil)
	}
	return types.TupleValueMust(elemTypes, rows)
}

func TestCSVEncode_BasicAutoHeaders(t *testing.T) {
	rows := makeRowList(
		makeRow(map[string]attr.Value{
			"name":  types.StringValue("alice"),
			"email": types.StringValue("alice@example.com"),
		}),
		makeRow(map[string]attr.Value{
			"name":  types.StringValue("bob"),
			"email": types.StringValue("bob@example.com"),
		}),
	)

	result, err := runCSVEncode(t, rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Headers should be sorted alphabetically.
	expected := "email,name\nalice@example.com,alice\nbob@example.com,bob\n"
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestCSVEncode_ExplicitColumns(t *testing.T) {
	rows := makeRowList(
		makeRow(map[string]attr.Value{
			"name":  types.StringValue("alice"),
			"email": types.StringValue("alice@example.com"),
			"role":  types.StringValue("admin"),
		}),
	)

	colsList := types.TupleValueMust(
		[]attr.Type{types.StringType, types.StringType, types.StringType},
		[]attr.Value{types.StringValue("name"), types.StringValue("email"), types.StringValue("role")},
	)
	opts := types.ObjectValueMust(
		map[string]attr.Type{"columns": colsList.Type(nil)},
		map[string]attr.Value{"columns": colsList},
	)

	result, err := runCSVEncode(t, rows, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "name,email,role\nalice,alice@example.com,admin\n"
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestCSVEncode_NoHeader(t *testing.T) {
	rows := makeRowList(
		makeRow(map[string]attr.Value{
			"a": types.StringValue("1"),
			"b": types.StringValue("2"),
		}),
	)

	colsList := types.TupleValueMust(
		[]attr.Type{types.StringType, types.StringType},
		[]attr.Value{types.StringValue("a"), types.StringValue("b")},
	)
	opts := types.ObjectValueMust(
		map[string]attr.Type{
			"columns":   colsList.Type(nil),
			"no_header": types.BoolType,
		},
		map[string]attr.Value{
			"columns":   colsList,
			"no_header": types.BoolValue(true),
		},
	)

	result, err := runCSVEncode(t, rows, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "1,2\n"
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestCSVEncode_TypeConversion(t *testing.T) {
	rows := makeRowList(
		makeRow(map[string]attr.Value{
			"name":    types.StringValue("alice"),
			"count":   types.NumberValue(big.NewFloat(42)),
			"ratio":   types.NumberValue(big.NewFloat(3.14)),
			"active":  types.BoolValue(true),
			"deleted": types.BoolValue(false),
		}),
	)

	colsList := types.TupleValueMust(
		[]attr.Type{types.StringType, types.StringType, types.StringType, types.StringType, types.StringType},
		[]attr.Value{
			types.StringValue("name"),
			types.StringValue("count"),
			types.StringValue("ratio"),
			types.StringValue("active"),
			types.StringValue("deleted"),
		},
	)
	opts := types.ObjectValueMust(
		map[string]attr.Type{"columns": colsList.Type(nil)},
		map[string]attr.Value{"columns": colsList},
	)

	result, err := runCSVEncode(t, rows, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "name,count,ratio,active,deleted\nalice,42,3.14,true,false\n"
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestCSVEncode_NullValues(t *testing.T) {
	rows := makeRowList(
		makeRow(map[string]attr.Value{
			"name":  types.StringValue("alice"),
			"email": types.StringNull(),
		}),
	)

	result, err := runCSVEncode(t, rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "email,name\n,alice\n"
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestCSVEncode_MissingKeys(t *testing.T) {
	// Rows with different keys — missing keys become empty cells.
	rows := makeRowList(
		makeRow(map[string]attr.Value{
			"name": types.StringValue("alice"),
			"role": types.StringValue("admin"),
		}),
		makeRow(map[string]attr.Value{
			"name":  types.StringValue("bob"),
			"email": types.StringValue("bob@example.com"),
		}),
	)

	result, err := runCSVEncode(t, rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Auto-detected columns: email, name, role (sorted).
	// Row 1: alice has no email, bob has no role.
	expected := "email,name,role\n,alice,admin\nbob@example.com,bob,\n"
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestCSVEncode_EmptyRows(t *testing.T) {
	rows := types.TupleValueMust([]attr.Type{}, []attr.Value{})

	result, err := runCSVEncode(t, rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "" {
		t.Errorf("expected empty string for empty rows, got %q", result)
	}
}

func TestCSVEncode_SpecialCharacters(t *testing.T) {
	// Values with commas, quotes, and newlines should be properly escaped.
	rows := makeRowList(
		makeRow(map[string]attr.Value{
			"name":  types.StringValue("O'Brien, James"),
			"quote": types.StringValue(`She said "hello"`),
			"bio":   types.StringValue("line1\nline2"),
		}),
	)

	result, err := runCSVEncode(t, rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// csv.Writer handles quoting automatically.
	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}
	// Verify the comma-containing value is quoted.
	if !contains(result, `"O'Brien, James"`) {
		t.Errorf("expected quoted value with comma in output:\n%s", result)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestCSVEncode_LossyRoundTrip(t *testing.T) {
	// Demonstrate that encode produces output compatible with Terraform's csvdecode,
	// but types are flattened to strings (known lossiness).
	rows := makeRowList(
		makeRow(map[string]attr.Value{
			"name":   types.StringValue("alice"),
			"count":  types.NumberValue(big.NewFloat(42)),
			"active": types.BoolValue(true),
		}),
	)

	result, err := runCSVEncode(t, rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The output is valid CSV that csvdecode can parse, but all values
	// will come back as strings: "42" not 42, "true" not true.
	// This is expected — CSV has no type system.
	expected := "active,count,name\ntrue,42,alice\n"
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestCSVEncode_TooManyOptions(t *testing.T) {
	rows := types.TupleValueMust([]attr.Type{}, []attr.Value{})
	emptyObj := types.ObjectValueMust(map[string]attr.Type{}, map[string]attr.Value{})

	_, err := runCSVEncode(t, rows, emptyObj, emptyObj)
	if err == nil {
		t.Fatal("expected error for too many options args")
	}
}

func TestCSVEncode_NotAList(t *testing.T) {
	_, err := runCSVEncode(t, types.StringValue("not a list"))
	if err == nil {
		t.Fatal("expected error for non-list input")
	}
}
