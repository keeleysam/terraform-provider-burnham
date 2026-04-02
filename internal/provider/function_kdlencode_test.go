package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runKDLEncode(t *testing.T, value attr.Value, opts ...attr.Value) (string, *function.FuncError) {
	t.Helper()
	f := &KDLEncodeFunction{}

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

func makeKDLNode(name string, args []attr.Value, props map[string]attr.Value, children []attr.Value) attr.Value {
	argTypes := make([]attr.Type, len(args))
	for i, a := range args {
		argTypes[i] = a.Type(nil)
	}
	argsTuple := types.TupleValueMust(argTypes, args)

	propTypes := make(map[string]attr.Type, len(props))
	for k, v := range props {
		propTypes[k] = v.Type(nil)
	}
	propsObj := types.ObjectValueMust(propTypes, props)

	childTypes := make([]attr.Type, len(children))
	for i, c := range children {
		childTypes[i] = c.Type(nil)
	}
	childrenTuple := types.TupleValueMust(childTypes, children)

	attrTypes := map[string]attr.Type{
		"name":     types.StringType,
		"args":     argsTuple.Type(nil),
		"props":    propsObj.Type(nil),
		"children": childrenTuple.Type(nil),
	}
	attrValues := map[string]attr.Value{
		"name":     types.StringValue(name),
		"args":     argsTuple,
		"props":    propsObj,
		"children": childrenTuple,
	}
	return types.ObjectValueMust(attrTypes, attrValues)
}

func TestKDLEncode_Basic(t *testing.T) {
	node := makeKDLNode("title", []attr.Value{types.StringValue("Hello")}, nil, nil)
	nodes := types.TupleValueMust(
		[]attr.Type{node.Type(nil)},
		[]attr.Value{node},
	)

	result, err := runKDLEncode(t, nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "title") {
		t.Errorf("expected 'title' in output:\n%s", result)
	}
	if !strings.Contains(result, "Hello") {
		t.Errorf("expected 'Hello' in output:\n%s", result)
	}
}

func TestKDLEncode_WithProps(t *testing.T) {
	node := makeKDLNode("author", []attr.Value{types.StringValue("Alex")},
		map[string]attr.Value{"email": types.StringValue("alex@example.com")}, nil)
	nodes := types.TupleValueMust([]attr.Type{node.Type(nil)}, []attr.Value{node})

	result, err := runKDLEncode(t, nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "email") {
		t.Errorf("expected 'email' property in output:\n%s", result)
	}
}

func TestKDLEncode_V1(t *testing.T) {
	node := makeKDLNode("key", []attr.Value{types.StringValue("val")},
		map[string]attr.Value{"prop": types.StringValue("test")}, nil)
	nodes := types.TupleValueMust([]attr.Type{node.Type(nil)}, []attr.Value{node})

	opts := types.ObjectValueMust(
		map[string]attr.Type{"version": types.StringType},
		map[string]attr.Value{"version": types.StringValue("v1")},
	)

	result, err := runKDLEncode(t, nodes, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// V1 should quote property values.
	if !strings.Contains(result, `prop="test"`) {
		t.Errorf("expected v1 quoted property in output:\n%s", result)
	}
}

func TestKDLEncode_RoundTrip(t *testing.T) {
	input := "title \"Hello, World\"\nauthor \"Alex\" email=\"alex@example.com\"\n"

	decoded, decErr := runKDLDecode(t, input)
	if decErr != nil {
		t.Fatalf("decode error: %v", decErr)
	}

	encoded, encErr := runKDLEncode(t, decoded.UnderlyingValue())
	if encErr != nil {
		t.Fatalf("encode error: %v", encErr)
	}

	if !strings.Contains(encoded, "title") || !strings.Contains(encoded, "Hello, World") {
		t.Errorf("expected title in round-trip:\n%s", encoded)
	}
	if !strings.Contains(encoded, "author") || !strings.Contains(encoded, "Alex") {
		t.Errorf("expected author in round-trip:\n%s", encoded)
	}
}

func TestKDLEncode_NotAList(t *testing.T) {
	_, err := runKDLEncode(t, types.StringValue("not a list"))
	if err == nil {
		t.Fatal("expected error for non-list input")
	}
}
