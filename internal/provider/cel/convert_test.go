package cel

import (
	"encoding/json"
	"math/big"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestTerraformToNode(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{"ident": types.StringType},
		map[string]attr.Value{"ident": types.StringValue("device.os_type")},
	)
	tuple := types.TupleValueMust(
		[]attr.Type{types.StringType, types.StringType},
		[]attr.Value{types.StringValue("US"), types.StringValue("CA")},
	)

	cases := []struct {
		name string
		in   attr.Value
		want any
	}{
		{"string", types.StringValue("US"), "US"},
		{"bool", types.BoolValue(true), true},
		{"number int", types.NumberValue(big.NewFloat(19)), json.Number("19")},
		{"null", types.StringNull(), nil},
		{"tuple", tuple, []any{"US", "CA"}},
		{"object single key", obj, map[string]any{"ident": "device.os_type"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := terraformToNode(tc.in)
			if err != nil {
				t.Fatalf("terraformToNode error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("terraformToNode = %#v, want %#v", got, tc.want)
			}
		})
	}
}

// End-to-end through the framework types into a CEL string.
func TestEncodeValueEndToEnd(t *testing.T) {
	node := types.ObjectValueMust(
		map[string]attr.Type{"==": types.TupleType{ElemTypes: []attr.Type{
			types.ObjectType{AttrTypes: map[string]attr.Type{"ident": types.StringType}},
			types.StringType,
		}}},
		map[string]attr.Value{"==": types.TupleValueMust(
			[]attr.Type{
				types.ObjectType{AttrTypes: map[string]attr.Type{"ident": types.StringType}},
				types.StringType,
			},
			[]attr.Value{
				types.ObjectValueMust(map[string]attr.Type{"ident": types.StringType}, map[string]attr.Value{"ident": types.StringValue("a")}),
				types.StringValue("b"),
			},
		)},
	)
	n, err := terraformToNode(node)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	got, err := Encode(n)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if got != `a == "b"` {
		t.Fatalf("got %q, want %q", got, `a == "b"`)
	}
}
