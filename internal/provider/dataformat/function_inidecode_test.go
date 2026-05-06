package dataformat

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runINIDecode(t *testing.T, input string) (types.Dynamic, *function.FuncError) {
	t.Helper()
	f := &INIDecodeFunction{}

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

func TestINIDecode_BasicSections(t *testing.T) {
	input := `[database]
host = localhost
port = 5432

[server]
address = 0.0.0.0
`

	result, err := runINIDecode(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj := result.UnderlyingValue().(types.Object)
	attrs := obj.Attributes()

	db := attrs["database"].(types.Object)
	host := db.Attributes()["host"].(types.String).ValueString()
	if host != "localhost" {
		t.Errorf("expected host='localhost', got %q", host)
	}
	port := db.Attributes()["port"].(types.String).ValueString()
	if port != "5432" {
		t.Errorf("expected port='5432', got %q", port)
	}

	srv := attrs["server"].(types.Object)
	addr := srv.Attributes()["address"].(types.String).ValueString()
	if addr != "0.0.0.0" {
		t.Errorf("expected address='0.0.0.0', got %q", addr)
	}
}

func TestINIDecode_GlobalKeys(t *testing.T) {
	input := `key = global_value

[section]
other = thing
`

	result, err := runINIDecode(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj := result.UnderlyingValue().(types.Object)
	attrs := obj.Attributes()

	global := attrs[""].(types.Object)
	val := global.Attributes()["key"].(types.String).ValueString()
	if val != "global_value" {
		t.Errorf("expected key='global_value', got %q", val)
	}

	section := attrs["section"].(types.Object)
	other := section.Attributes()["other"].(types.String).ValueString()
	if other != "thing" {
		t.Errorf("expected other='thing', got %q", other)
	}
}

func TestINIDecode_Comments(t *testing.T) {
	input := `; this is a comment
# this is also a comment
[section]
key = value
`

	result, err := runINIDecode(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj := result.UnderlyingValue().(types.Object)
	section := obj.Attributes()["section"].(types.Object)
	val := section.Attributes()["key"].(types.String).ValueString()
	if val != "value" {
		t.Errorf("expected key='value', got %q", val)
	}
}

func TestINIDecode_EmptyString(t *testing.T) {
	// Empty string should parse as an empty INI (just a default section).
	result, err := runINIDecode(t, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsNull() {
		t.Fatal("expected non-null result")
	}
}

func TestINIDecode_EmptySection(t *testing.T) {
	input := `[empty]

[notempty]
key = val
`

	result, err := runINIDecode(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj := result.UnderlyingValue().(types.Object)
	empty := obj.Attributes()["empty"].(types.Object)
	if len(empty.Attributes()) != 0 {
		t.Errorf("expected empty section, got %d keys", len(empty.Attributes()))
	}
}
