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

func runYAMLEncode(t *testing.T, value attr.Value, opts ...attr.Value) (string, *function.FuncError) {
	t.Helper()
	f := &YAMLEncodeFunction{}

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

func makeYAMLOpts(kv map[string]attr.Value) attr.Value {
	attrTypes := make(map[string]attr.Type, len(kv))
	for k, v := range kv {
		attrTypes[k] = v.Type(nil)
	}
	return types.ObjectValueMust(attrTypes, kv)
}

// ─── Default behavior ────────────────────────────────────────────

func TestYAMLEncode_BasicMap(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{"name": types.StringType, "version": types.NumberType},
		map[string]attr.Value{"name": types.StringValue("test"), "version": types.NumberValue(big.NewFloat(1))},
	)

	result, err := runYAMLEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "name: test") {
		t.Errorf("expected 'name: test' in output:\n%s", result)
	}
	if !strings.Contains(result, "version: 1") {
		t.Errorf("expected 'version: 1' in output:\n%s", result)
	}
	// Block style: no { } braces.
	if strings.Contains(result, "{") {
		t.Errorf("expected block style (no braces):\n%s", result)
	}
}

func TestYAMLEncode_NestedMap(t *testing.T) {
	inner := types.ObjectValueMust(
		map[string]attr.Type{"host": types.StringType},
		map[string]attr.Value{"host": types.StringValue("localhost")},
	)
	obj := types.ObjectValueMust(
		map[string]attr.Type{"database": inner.Type(nil)},
		map[string]attr.Value{"database": inner},
	)

	result, err := runYAMLEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "database:\n") {
		t.Errorf("expected nested block style:\n%s", result)
	}
	if !strings.Contains(result, "  host: localhost") {
		t.Errorf("expected indented nested key:\n%s", result)
	}
}

func TestYAMLEncode_List(t *testing.T) {
	arr := types.TupleValueMust(
		[]attr.Type{types.StringType, types.StringType},
		[]attr.Value{types.StringValue("a"), types.StringValue("b")},
	)
	obj := types.ObjectValueMust(
		map[string]attr.Type{"items": arr.Type(nil)},
		map[string]attr.Value{"items": arr},
	)

	result, err := runYAMLEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "- a") {
		t.Errorf("expected block list with '- a':\n%s", result)
	}
}

func TestYAMLEncode_MultilineString(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{"script": types.StringType},
		map[string]attr.Value{"script": types.StringValue("#!/bin/bash\necho hello\nexit 0\n")},
	)

	result, err := runYAMLEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "|") {
		t.Errorf("expected literal block scalar '|' for multi-line string:\n%s", result)
	}
	if strings.Contains(result, "\\n") {
		t.Errorf("multi-line string should not contain literal \\n:\n%s", result)
	}
}

func TestYAMLEncode_Bool(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{"enabled": types.BoolType, "debug": types.BoolType},
		map[string]attr.Value{"enabled": types.BoolValue(true), "debug": types.BoolValue(false)},
	)

	result, err := runYAMLEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "enabled: true") {
		t.Errorf("expected 'enabled: true':\n%s", result)
	}
	if !strings.Contains(result, "debug: false") {
		t.Errorf("expected 'debug: false':\n%s", result)
	}
}

func TestYAMLEncode_Float(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{"ratio": types.NumberType},
		map[string]attr.Value{"ratio": types.NumberValue(big.NewFloat(3.14))},
	)

	result, err := runYAMLEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "ratio: 3.14") {
		t.Errorf("expected 'ratio: 3.14':\n%s", result)
	}
}

func TestYAMLEncode_Null(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{"empty": types.StringType},
		map[string]attr.Value{"empty": types.StringNull()},
	)

	result, err := runYAMLEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "empty: null") {
		t.Errorf("expected 'empty: null':\n%s", result)
	}
}

func TestYAMLEncode_SortedKeys(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{"zebra": types.StringType, "apple": types.StringType},
		map[string]attr.Value{"zebra": types.StringValue("z"), "apple": types.StringValue("a")},
	)

	result, err := runYAMLEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	appleIdx := strings.Index(result, "apple:")
	zebraIdx := strings.Index(result, "zebra:")
	if appleIdx > zebraIdx {
		t.Errorf("expected sorted keys (apple before zebra):\n%s", result)
	}
}

// ─── Options ─────────────────────────────────────────────────────

func TestYAMLEncode_Indent4(t *testing.T) {
	inner := types.ObjectValueMust(
		map[string]attr.Type{"key": types.StringType},
		map[string]attr.Value{"key": types.StringValue("val")},
	)
	obj := types.ObjectValueMust(
		map[string]attr.Type{"outer": inner.Type(nil)},
		map[string]attr.Value{"outer": inner},
	)

	opts := makeYAMLOpts(map[string]attr.Value{
		"indent": types.NumberValue(big.NewFloat(4)),
	})

	result, err := runYAMLEncode(t, obj, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "    key: val") {
		t.Errorf("expected 4-space indent:\n%s", result)
	}
}

func TestYAMLEncode_FlowAll(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{"a": types.StringType},
		map[string]attr.Value{"a": types.StringValue("b")},
	)

	opts := makeYAMLOpts(map[string]attr.Value{
		"flow_level": types.NumberValue(big.NewFloat(-1)),
	})

	result, err := runYAMLEncode(t, obj, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "{") {
		t.Errorf("expected flow style with braces:\n%s", result)
	}
}

func TestYAMLEncode_MultilineFolded(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{"text": types.StringType},
		map[string]attr.Value{"text": types.StringValue("line1\nline2\n")},
	)

	opts := makeYAMLOpts(map[string]attr.Value{
		"multiline": types.StringValue("folded"),
	})

	result, err := runYAMLEncode(t, obj, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, ">") {
		t.Errorf("expected folded style '>':\n%s", result)
	}
}

func TestYAMLEncode_QuoteDouble(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{"name": types.StringType},
		map[string]attr.Value{"name": types.StringValue("test")},
	)

	opts := makeYAMLOpts(map[string]attr.Value{
		"quote_style": types.StringValue("double"),
	})

	result, err := runYAMLEncode(t, obj, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, `"test"`) {
		t.Errorf("expected double-quoted string:\n%s", result)
	}
}

func TestYAMLEncode_QuoteSingle(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{"name": types.StringType},
		map[string]attr.Value{"name": types.StringValue("test")},
	)

	opts := makeYAMLOpts(map[string]attr.Value{
		"quote_style": types.StringValue("single"),
	})

	result, err := runYAMLEncode(t, obj, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "'test'") {
		t.Errorf("expected single-quoted string:\n%s", result)
	}
}

func TestYAMLEncode_NullTilde(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{"empty": types.StringType},
		map[string]attr.Value{"empty": types.StringNull()},
	)

	opts := makeYAMLOpts(map[string]attr.Value{
		"null_value": types.StringValue("~"),
	})

	result, err := runYAMLEncode(t, obj, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "empty: ~") {
		t.Errorf("expected 'empty: ~':\n%s", result)
	}
}

func TestYAMLEncode_SortKeysFalse(t *testing.T) {
	// With sort_keys=false, keys should appear in... well, Go map order is random,
	// but at least we verify it doesn't crash.
	obj := types.ObjectValueMust(
		map[string]attr.Type{"z": types.StringType, "a": types.StringType},
		map[string]attr.Value{"z": types.StringValue("1"), "a": types.StringValue("2")},
	)

	opts := makeYAMLOpts(map[string]attr.Value{
		"sort_keys": types.BoolValue(false),
	})

	result, err := runYAMLEncode(t, obj, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Just verify both keys are present.
	if !strings.Contains(result, "z:") || !strings.Contains(result, "a:") {
		t.Errorf("expected both keys in output:\n%s", result)
	}
}

// ─── Comments ────────────────────────────────────────────────────

func TestYAMLEncode_Comments(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{"apiVersion": types.StringType, "kind": types.StringType},
		map[string]attr.Value{"apiVersion": types.StringValue("v1"), "kind": types.StringValue("ConfigMap")},
	)

	comments := types.ObjectValueMust(
		map[string]attr.Type{"apiVersion": types.StringType},
		map[string]attr.Value{"apiVersion": types.StringValue("Kubernetes API version")},
	)

	opts := makeYAMLOpts(map[string]attr.Value{
		"comments": comments,
	})

	result, err := runYAMLEncode(t, obj, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "# Kubernetes API version") {
		t.Errorf("expected comment in output:\n%s", result)
	}
}

func TestYAMLEncode_NestedComments(t *testing.T) {
	inner := types.ObjectValueMust(
		map[string]attr.Type{"name": types.StringType},
		map[string]attr.Value{"name": types.StringValue("test")},
	)
	obj := types.ObjectValueMust(
		map[string]attr.Type{"metadata": inner.Type(nil)},
		map[string]attr.Value{"metadata": inner},
	)

	innerComments := types.ObjectValueMust(
		map[string]attr.Type{"name": types.StringType},
		map[string]attr.Value{"name": types.StringValue("Resource name")},
	)
	comments := types.ObjectValueMust(
		map[string]attr.Type{"metadata": innerComments.Type(nil)},
		map[string]attr.Value{"metadata": innerComments},
	)

	opts := makeYAMLOpts(map[string]attr.Value{"comments": comments})

	result, err := runYAMLEncode(t, obj, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "# Resource name") {
		t.Errorf("expected nested comment:\n%s", result)
	}
}

// ─── Real-world ──────────────────────────────────────────────────

func TestYAMLEncode_KubernetesConfigMap(t *testing.T) {
	data := types.ObjectValueMust(
		map[string]attr.Type{"startup.sh": types.StringType, "config.ini": types.StringType},
		map[string]attr.Value{
			"startup.sh": types.StringValue("#!/bin/bash\nset -e\necho Starting...\n./run-app\n"),
			"config.ini": types.StringValue("[server]\nport=8080\n"),
		},
	)
	metadata := types.ObjectValueMust(
		map[string]attr.Type{"name": types.StringType, "namespace": types.StringType},
		map[string]attr.Value{"name": types.StringValue("app-config"), "namespace": types.StringValue("production")},
	)
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"apiVersion": types.StringType,
			"kind":       types.StringType,
			"metadata":   metadata.Type(nil),
			"data":       data.Type(nil),
		},
		map[string]attr.Value{
			"apiVersion": types.StringValue("v1"),
			"kind":       types.StringValue("ConfigMap"),
			"metadata":   metadata,
			"data":       data,
		},
	)

	result, err := runYAMLEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Multi-line strings should use literal block scalars.
	if !strings.Contains(result, "|") {
		t.Errorf("expected literal block scalar for multi-line strings:\n%s", result)
	}
	// First line should not be flow style.
	if strings.HasPrefix(strings.TrimSpace(result), "{") {
		t.Errorf("expected block style at top level:\n%s", result)
	}
	// Key structure should be present.
	if !strings.Contains(result, "apiVersion: v1") {
		t.Errorf("expected apiVersion:\n%s", result)
	}
	if !strings.Contains(result, "kind: ConfigMap") {
		t.Errorf("expected kind:\n%s", result)
	}
}

// ─── Dedupe ──────────────────────────────────────────────────────

func TestYAMLEncode_Dedupe(t *testing.T) {
	shared := types.ObjectValueMust(
		map[string]attr.Type{"host": types.StringType, "port": types.NumberType},
		map[string]attr.Value{"host": types.StringValue("localhost"), "port": types.NumberValue(big.NewFloat(5432))},
	)
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"dev":     shared.Type(nil),
			"staging": shared.Type(nil),
		},
		map[string]attr.Value{
			"dev":     shared,
			"staging": shared,
		},
	)

	opts := makeYAMLOpts(map[string]attr.Value{
		"dedupe": types.BoolValue(true),
	})

	result, err := runYAMLEncode(t, obj, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "&") {
		t.Errorf("expected anchor (&) in deduped output:\n%s", result)
	}
	if !strings.Contains(result, "*") {
		t.Errorf("expected alias (*) in deduped output:\n%s", result)
	}
}

func TestYAMLEncode_DedupeOff(t *testing.T) {
	shared := types.ObjectValueMust(
		map[string]attr.Type{"host": types.StringType},
		map[string]attr.Value{"host": types.StringValue("localhost")},
	)
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"a": shared.Type(nil),
			"b": shared.Type(nil),
		},
		map[string]attr.Value{
			"a": shared,
			"b": shared,
		},
	)

	// Default (no dedupe option) — should NOT have anchors.
	result, err := runYAMLEncode(t, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(result, "&") || strings.Contains(result, "*") {
		t.Errorf("expected no anchors/aliases without dedupe:\n%s", result)
	}
}

func TestYAMLEncode_DedupeNoDuplicates(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{
			"a": types.StringType,
			"b": types.StringType,
		},
		map[string]attr.Value{
			"a": types.StringValue("unique1"),
			"b": types.StringValue("unique2"),
		},
	)

	opts := makeYAMLOpts(map[string]attr.Value{
		"dedupe": types.BoolValue(true),
	})

	result, err := runYAMLEncode(t, obj, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No duplicates means no anchors.
	if strings.Contains(result, "&") {
		t.Errorf("expected no anchors when nothing is duplicated:\n%s", result)
	}
}

func TestYAMLEncode_TooManyOpts(t *testing.T) {
	obj := types.ObjectValueMust(map[string]attr.Type{}, map[string]attr.Value{})
	empty := types.ObjectValueMust(map[string]attr.Type{}, map[string]attr.Value{})
	_, err := runYAMLEncode(t, obj, empty, empty)
	if err == nil {
		t.Fatal("expected error for too many options")
	}
}
