package dataformat

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runKDLDecode(t *testing.T, input string) (types.Dynamic, *function.FuncError) {
	t.Helper()
	f := &KDLDecodeFunction{}

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

func TestKDLDecode_Basic(t *testing.T) {
	input := `title "Hello, World"`

	result, err := runKDLDecode(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tuple := result.UnderlyingValue().(types.Tuple)
	if len(tuple.Elements()) != 1 {
		t.Fatalf("expected 1 node, got %d", len(tuple.Elements()))
	}

	node := tuple.Elements()[0].(types.Object)
	name := node.Attributes()["name"].(types.String).ValueString()
	if name != "title" {
		t.Errorf("expected name='title', got %q", name)
	}
}

func TestKDLDecode_WithProps(t *testing.T) {
	input := `author "Alex" email="alex@example.com"`

	result, err := runKDLDecode(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tuple := result.UnderlyingValue().(types.Tuple)
	node := tuple.Elements()[0].(types.Object)
	props := node.Attributes()["props"].(types.Object)
	email := props.Attributes()["email"].(types.String).ValueString()
	if email != "alex@example.com" {
		t.Errorf("expected email='alex@example.com', got %q", email)
	}
}

func TestKDLDecode_WithChildren(t *testing.T) {
	input := "parent {\n  child \"value\"\n}\n"

	result, err := runKDLDecode(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tuple := result.UnderlyingValue().(types.Tuple)
	node := tuple.Elements()[0].(types.Object)
	children := node.Attributes()["children"].(types.Tuple)
	if len(children.Elements()) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children.Elements()))
	}
}

func TestKDLDecode_Invalid(t *testing.T) {
	_, err := runKDLDecode(t, "{{{{invalid")
	if err == nil {
		t.Fatal("expected error for invalid KDL")
	}
}

func TestKDLDecode_MultipleNodes(t *testing.T) {
	input := "node1 \"a\"\nnode2 \"b\"\nnode3 \"c\"\n"

	result, err := runKDLDecode(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tuple := result.UnderlyingValue().(types.Tuple)
	if len(tuple.Elements()) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(tuple.Elements()))
	}
}
