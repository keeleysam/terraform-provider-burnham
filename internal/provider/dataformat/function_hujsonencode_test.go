package dataformat

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runHuJSONEncode(t *testing.T, value attr.Value, opts ...attr.Value) (string, *function.FuncError) {
	t.Helper()
	f := &HuJSONEncodeFunction{}

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

func TestHuJSONEncode_LargeObject(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"name":        types.StringType,
			"enabled":     types.BoolType,
			"description": types.StringType,
			"version":     types.NumberType,
		},
		map[string]attr.Value{
			"name":        types.StringValue("test-profile"),
			"enabled":     types.BoolValue(true),
			"description": types.StringValue("A longer description for testing"),
			"version":     types.NumberValue(big.NewFloat(42)),
		},
	)

	result, err := runHuJSONEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, ",\n") {
		t.Errorf("expected trailing commas in multi-line output:\n%s", result)
	}

	if !strings.Contains(result, "\t") {
		t.Errorf("expected tab indentation in output:\n%s", result)
	}

	if strings.Contains(result, "//") {
		t.Errorf("should not contain injected comment:\n%s", result)
	}
}

func TestHuJSONEncode_SmallObject_Default(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"a": types.NumberType,
		},
		map[string]attr.Value{
			"a": types.NumberValue(big.NewFloat(1)),
		},
	)

	result, err := runHuJSONEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Default is the always-expanded layout.
	if result != "{\n\t\"a\": 1,\n}" {
		t.Errorf("expected always-expanded default output, got:\n%q", result)
	}
}

func TestHuJSONEncode_SmallObject_Compact(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"a": types.NumberType,
		},
		map[string]attr.Value{
			"a": types.NumberValue(big.NewFloat(1)),
		},
	)

	result, err := runHuJSONEncode(t, obj, makeBoolOpts("compact", true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "{\"a\": 1}\n" {
		t.Errorf("expected compact single-line output, got:\n%q", result)
	}
}

func TestHuJSONEncode_CustomIndent(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"name":        types.StringType,
			"enabled":     types.BoolType,
			"description": types.StringType,
			"version":     types.NumberType,
		},
		map[string]attr.Value{
			"name":        types.StringValue("test-profile"),
			"enabled":     types.BoolValue(true),
			"description": types.StringValue("A longer description for testing"),
			"version":     types.NumberValue(big.NewFloat(42)),
		},
	)

	result, err := runHuJSONEncode(t, obj, makeIndentOpts("  "))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(result, "\t") {
		t.Errorf("expected no tabs in output with 2-space indent:\n%s", result)
	}
	if !strings.Contains(result, "  ") {
		t.Errorf("expected 2-space indentation in output:\n%s", result)
	}
}

func TestHuJSONEncode_NestedArray(t *testing.T) {
	innerArr := types.TupleValueMust(
		[]attr.Type{types.StringType, types.StringType, types.StringType},
		[]attr.Value{
			types.StringValue("user1@example.com"),
			types.StringValue("user2@example.com"),
			types.StringValue("user3@example.com"),
		},
	)

	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"members": innerArr.Type(nil),
		},
		map[string]attr.Value{
			"members": innerArr,
		},
	)

	result, err := runHuJSONEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "user1@example.com") {
		t.Errorf("expected user1 in output:\n%s", result)
	}
}

func TestHuJSONEncode_NoInjectedComment(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"key": types.StringType,
		},
		map[string]attr.Value{
			"key": types.StringValue("value"),
		},
	)

	result, err := runHuJSONEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.HasPrefix(result, "//") {
		t.Errorf("output should not start with injected comment:\n%s", result)
	}
}

func TestHuJSONEncode_TooManyOpts(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{},
		map[string]attr.Value{},
	)
	empty := types.ObjectValueMust(map[string]attr.Type{}, map[string]attr.Value{})
	_, err := runHuJSONEncode(t, obj, empty, empty)
	if err == nil {
		t.Fatal("expected error for too many options args")
	}
}

// ─── Comment tests ───────────────────────────────────────────────

func makeCommentsOpts(comments attr.Value) attr.Value {
	return types.ObjectValueMust(
		map[string]attr.Type{"comments": comments.Type(nil)},
		map[string]attr.Value{"comments": comments},
	)
}

func TestHuJSONEncode_SingleLineComment(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"acls":   types.StringType,
			"groups": types.StringType,
		},
		map[string]attr.Value{
			"acls":   types.StringValue("data"),
			"groups": types.StringValue("data"),
		},
	)

	comments := types.ObjectValueMust(
		map[string]attr.Type{
			"acls":   types.StringType,
			"groups": types.StringType,
		},
		map[string]attr.Value{
			"acls":   types.StringValue("Network ACLs"),
			"groups": types.StringValue("Group definitions"),
		},
	)

	result, err := runHuJSONEncode(t, obj, makeCommentsOpts(comments))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "// Network ACLs") {
		t.Errorf("expected // Network ACLs in output:\n%s", result)
	}
	if !strings.Contains(result, "// Group definitions") {
		t.Errorf("expected // Group definitions in output:\n%s", result)
	}
}

func TestHuJSONEncode_MultiLineComment(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"config": types.StringType,
		},
		map[string]attr.Value{
			"config": types.StringValue("data"),
		},
	)

	comments := types.ObjectValueMust(
		map[string]attr.Type{
			"config": types.StringType,
		},
		map[string]attr.Value{
			"config": types.StringValue("This is a\nmulti-line comment"),
		},
	)

	result, err := runHuJSONEncode(t, obj, makeCommentsOpts(comments))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "/*") || !strings.Contains(result, "*/") {
		t.Errorf("expected /* */ block comment in output:\n%s", result)
	}
}

func TestHuJSONEncode_NestedComments(t *testing.T) {
	inner := types.ObjectValueMust(
		map[string]attr.Type{
			"host": types.StringType,
			"port": types.StringType,
		},
		map[string]attr.Value{
			"host": types.StringValue("localhost"),
			"port": types.StringValue("5432"),
		},
	)

	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"database": inner.Type(nil),
		},
		map[string]attr.Value{
			"database": inner,
		},
	)

	innerComments := types.ObjectValueMust(
		map[string]attr.Type{
			"host": types.StringType,
		},
		map[string]attr.Value{
			"host": types.StringValue("Database hostname"),
		},
	)

	comments := types.ObjectValueMust(
		map[string]attr.Type{
			"database": innerComments.Type(nil),
		},
		map[string]attr.Value{
			"database": innerComments,
		},
	)

	result, err := runHuJSONEncode(t, obj, makeCommentsOpts(comments))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "// Database hostname") {
		t.Errorf("expected nested comment in output:\n%s", result)
	}
}

func TestHuJSONEncode_CommentOnMissingKey(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"existing": types.StringType,
		},
		map[string]attr.Value{
			"existing": types.StringValue("data"),
		},
	)

	comments := types.ObjectValueMust(
		map[string]attr.Type{
			"nonexistent": types.StringType,
		},
		map[string]attr.Value{
			"nonexistent": types.StringValue("This key doesn't exist"),
		},
	)

	// Should not error — missing keys are silently skipped.
	result, err := runHuJSONEncode(t, obj, makeCommentsOpts(comments))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(result, "This key doesn't exist") {
		t.Errorf("comment for missing key should not appear in output:\n%s", result)
	}
}

func TestHuJSONEncode_EmptyComments(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{"a": types.StringType},
		map[string]attr.Value{"a": types.StringValue("b")},
	)

	emptyComments := types.ObjectValueMust(
		map[string]attr.Type{},
		map[string]attr.Value{},
	)

	result, err := runHuJSONEncode(t, obj, makeCommentsOpts(emptyComments))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(result, "//") || strings.Contains(result, "/*") {
		t.Errorf("empty comments should produce no comments:\n%s", result)
	}
}

func TestHuJSONEncode_CommentsWithIndent(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"key1": types.StringType,
			"key2": types.StringType,
		},
		map[string]attr.Value{
			"key1": types.StringValue("val1"),
			"key2": types.StringValue("val2"),
		},
	)

	comments := types.ObjectValueMust(
		map[string]attr.Type{"key1": types.StringType},
		map[string]attr.Value{"key1": types.StringValue("First key")},
	)

	opts := types.ObjectValueMust(
		map[string]attr.Type{
			"indent":   types.StringType,
			"comments": comments.Type(nil),
		},
		map[string]attr.Value{
			"indent":   types.StringValue("  "),
			"comments": comments,
		},
	)

	result, err := runHuJSONEncode(t, obj, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "// First key") {
		t.Errorf("expected comment in output:\n%s", result)
	}
	if strings.Contains(result, "\t") {
		t.Errorf("expected no tabs with 2-space indent:\n%s", result)
	}
}

func TestHuJSONEncode_CommentEscapesBlockClose(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{"key": types.StringType},
		map[string]attr.Value{"key": types.StringValue("val")},
	)

	// Multi-line comment containing */ which would break a /* */ block.
	comments := types.ObjectValueMust(
		map[string]attr.Type{"key": types.StringType},
		map[string]attr.Value{"key": types.StringValue("line1\nend: */\nline3")},
	)

	result, err := runHuJSONEncode(t, obj, makeCommentsOpts(comments))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The */ should be escaped so it doesn't break the block comment.
	if strings.Contains(result, "end: */") && !strings.Contains(result, "end: *\\/") {
		t.Errorf("expected */ to be escaped in block comment:\n%s", result)
	}
}
