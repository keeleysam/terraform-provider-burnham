package transform

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// knownObjectWithNestedUnknown builds { a = <unknown>, b = "x" }: the container is
// known, so core does not defer the call, but a nested value is unknown.
func knownObjectWithNestedUnknown() attr.Value {
	return types.ObjectValueMust(
		map[string]attr.Type{"a": types.StringType, "b": types.StringType},
		map[string]attr.Value{"a": types.StringUnknown(), "b": types.StringValue("x")},
	)
}

func TestJMESPathQuery_NestedUnknownReturnsUnknown(t *testing.T) {
	f := &JMESPathQueryFunction{}
	args := function.NewArgumentsData([]attr.Value{
		types.DynamicValue(knownObjectWithNestedUnknown()),
		types.StringValue("a"),
	})
	resp := &function.RunResponse{Result: function.NewResultData(types.DynamicNull())}
	f.Run(context.Background(), function.RunRequest{Arguments: args}, resp)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	if !resp.Result.Value().IsUnknown() {
		t.Fatalf("expected unknown result for nested unknown, got %#v", resp.Result.Value())
	}
}

func TestJQ_NestedUnknownReturnsUnknown(t *testing.T) {
	f := &JQFunction{}
	args := function.NewArgumentsData([]attr.Value{
		types.DynamicValue(knownObjectWithNestedUnknown()),
		types.StringValue("."),
		types.TupleValueMust([]attr.Type{}, []attr.Value{}),
	})
	resp := &function.RunResponse{Result: function.NewResultData(types.DynamicNull())}
	f.Run(context.Background(), function.RunRequest{Arguments: args}, resp)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	if !resp.Result.Value().IsUnknown() {
		t.Fatalf("expected unknown result for nested unknown, got %#v", resp.Result.Value())
	}
}

func TestJQ_UnknownVarReturnsUnknown(t *testing.T) {
	f := &JQFunction{}
	vars := types.ObjectValueMust(
		map[string]attr.Type{"tier": types.StringType},
		map[string]attr.Value{"tier": types.StringUnknown()},
	)
	opts := types.ObjectValueMust(
		map[string]attr.Type{"vars": vars.Type(context.Background())},
		map[string]attr.Value{"vars": vars},
	)
	args := function.NewArgumentsData([]attr.Value{
		types.DynamicValue(types.StringValue("data")),
		types.StringValue("."),
		types.TupleValueMust([]attr.Type{types.DynamicType}, []attr.Value{types.DynamicValue(opts)}),
	})
	resp := &function.RunResponse{Result: function.NewResultData(types.DynamicNull())}
	f.Run(context.Background(), function.RunRequest{Arguments: args}, resp)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	if !resp.Result.Value().IsUnknown() {
		t.Fatalf("expected unknown result for unknown option var, got %#v", resp.Result.Value())
	}
}
