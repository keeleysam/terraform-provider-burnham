package transform

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// runJSONata is the pure core of jsonata_query: it evaluates an expression
// against a value in the JSON value space (json.Number for numbers) and returns
// the result in the same space.

func jsonataBooks() map[string]interface{} {
	return map[string]interface{}{
		"books": []interface{}{
			map[string]interface{}{"title": "cheap", "price": json.Number("5")},
			map[string]interface{}{"title": "mid", "price": json.Number("10")},
			map[string]interface{}{"title": "dear", "price": json.Number("20")},
		},
	}
}

func TestRunJSONata_FieldAccess(t *testing.T) {
	got, err := runJSONata(context.Background(), jsonataBooks(), "books.title")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []interface{}{"cheap", "mid", "dear"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestRunJSONata_Filter(t *testing.T) {
	got, err := runJSONata(context.Background(), jsonataBooks(), "books[price>=10].title")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []interface{}{"mid", "dear"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestRunJSONata_ObjectConstruction(t *testing.T) {
	got, err := runJSONata(context.Background(), jsonataBooks(), "{'count': $count(books), 'total': $sum(books.price)}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]interface{}{"count": float64(3), "total": float64(35)}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestRunJSONata_Functions(t *testing.T) {
	got, err := runJSONata(context.Background(), nil, "$sum($map([1,2,3], function($v){$v*2}))")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != float64(12) {
		t.Errorf("got %#v, want 12", got)
	}
}

func TestRunJSONata_NumericArithmetic(t *testing.T) {
	data := map[string]interface{}{"a": json.Number("3"), "b": json.Number("4")}
	got, err := runJSONata(context.Background(), data, "a + b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != float64(7) {
		t.Errorf("got %#v, want 7", got)
	}
}

// TestRunJSONata_PassThroughPreservesPrecision documents that a number selected
// without any arithmetic passes through unchanged (as json.Number), so a big
// integer keeps full precision when it is merely extracted.
func TestRunJSONata_PassThroughPreservesPrecision(t *testing.T) {
	data := map[string]interface{}{"n": json.Number("9007199254740993")}
	got, err := runJSONata(context.Background(), data, "n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != json.Number("9007199254740993") {
		t.Errorf("got %#v, want json.Number(9007199254740993)", got)
	}
}

// TestRunJSONata_ArithmeticLosesPrecisionBeyond2p53 locks the documented number
// model: JSONata computes on IEEE-754 doubles, so an integer beyond 2^53 rounds
// once it takes part in arithmetic. 9007199254740993 (2^53 + 1) is not
// representable and becomes 9007199254740992. If a future change preserved
// precision through arithmetic, update the docs.
func TestRunJSONata_ArithmeticLosesPrecisionBeyond2p53(t *testing.T) {
	data := map[string]interface{}{"n": json.Number("9007199254740993")}
	got, err := runJSONata(context.Background(), data, "n + 0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != float64(9007199254740992) {
		t.Errorf("got %#v, want the float64-rounded 9007199254740992", got)
	}
}

// TestRunJSONata_KeyOrderIsDeterministic confirms order-sensitive builtins are
// stable: the input is decoded with keys sorted, so $keys never churns the plan.
func TestRunJSONata_KeyOrderIsDeterministic(t *testing.T) {
	data := map[string]interface{}{"z": json.Number("1"), "a": json.Number("2"), "m": json.Number("3")}
	got, err := runJSONata(context.Background(), data, "$keys($)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []interface{}{"a", "m", "z"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestRunJSONata_DeterminismGuard(t *testing.T) {
	for _, expr := range []string{"$now()", "$millis()", "$random()"} {
		_, err := runJSONata(context.Background(), nil, expr)
		if err == nil {
			t.Errorf("%s: expected a determinism error, got nil", expr)
			continue
		}
		if !strings.Contains(err.Error(), "disabled") {
			t.Errorf("%s: expected a clear determinism error, got %v", expr, err)
		}
	}
}

func TestRunJSONata_DeterminismGuardDoesNotAffectNormalExpressions(t *testing.T) {
	got, err := runJSONata(context.Background(), jsonataBooks(), "$count(books)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != float64(3) {
		t.Errorf("got %#v, want 3", got)
	}
}

// TestRunJSONata_NonFiniteRejected mirrors the rest of the transform package:
// a computed non-finite number (1/0 yields +Inf) is rejected on the way back to
// Terraform, not silently coerced.
func TestRunJSONata_NonFiniteRejected(t *testing.T) {
	got, err := runJSONata(context.Background(), nil, "1/0")
	if err != nil {
		t.Fatalf("unexpected eval error: %v", err)
	}
	if _, err := jsonToTerraform(got); err == nil {
		t.Fatalf("expected jsonToTerraform to reject non-finite result, got nil")
	} else if !strings.Contains(err.Error(), "non-finite") {
		t.Errorf("expected a non-finite error, got %v", err)
	}
}

func TestIsValidJSONata_Valid(t *testing.T) {
	for _, expr := range []string{"a.b.c", "books[price>10].title", "$sum([1,2,3])", "{'k': v}", "$now()"} {
		if !isValidJSONata(expr) {
			t.Errorf("expected %q to be valid", expr)
		}
	}
}

func TestIsValidJSONata_Malformed(t *testing.T) {
	for _, expr := range []string{"a[..", "{unclosed", "1 +"} {
		if isValidJSONata(expr) {
			t.Errorf("expected %q to be invalid", expr)
		}
	}
}

func TestIsValidJSONata_OversizedReturnsFalse(t *testing.T) {
	big := strings.Repeat("a", jsonataMaxInputBytes+1)
	if isValidJSONata(big) {
		t.Errorf("expected an oversized expression to be reported invalid, not panic or accept")
	}
}

func TestJSONataQuery_NestedUnknownReturnsUnknown(t *testing.T) {
	f := &JSONataQueryFunction{}
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

func TestJSONataQuery_NullInput(t *testing.T) {
	f := &JSONataQueryFunction{}
	args := function.NewArgumentsData([]attr.Value{
		types.DynamicValue(types.DynamicNull()),
		types.StringValue("$"),
	})
	resp := &function.RunResponse{Result: function.NewResultData(types.DynamicNull())}
	f.Run(context.Background(), function.RunRequest{Arguments: args}, resp)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	dyn, ok := resp.Result.Value().(types.Dynamic)
	if !ok {
		t.Fatalf("expected a dynamic result, got %T", resp.Result.Value())
	}
	if !dyn.IsNull() && !dyn.UnderlyingValue().IsNull() {
		t.Fatalf("expected null result for null input, got %#v", resp.Result.Value())
	}
}

func TestJSONataQuery_DeterminismGuardIsArgumentError(t *testing.T) {
	f := &JSONataQueryFunction{}
	args := function.NewArgumentsData([]attr.Value{
		types.DynamicValue(types.StringValue("x")),
		types.StringValue("$now()"),
	})
	resp := &function.RunResponse{Result: function.NewResultData(types.DynamicNull())}
	f.Run(context.Background(), function.RunRequest{Arguments: args}, resp)
	if resp.Error == nil {
		t.Fatalf("expected an error for $now(), got nil")
	}
}

func TestJSONataValidate_ReportsValidity(t *testing.T) {
	cases := map[string]bool{
		"a.b.c":  true,
		"a[..":   false,
		"$now()": true,
	}
	for expr, want := range cases {
		f := &JSONataValidateFunction{}
		args := function.NewArgumentsData([]attr.Value{types.StringValue(expr)})
		resp := &function.RunResponse{Result: function.NewResultData(types.BoolNull())}
		f.Run(context.Background(), function.RunRequest{Arguments: args}, resp)
		if resp.Error != nil {
			t.Fatalf("%q: unexpected error (validate must never fail the plan): %v", expr, resp.Error)
		}
		got, ok := resp.Result.Value().(types.Bool)
		if !ok {
			t.Fatalf("%q: expected a bool result, got %T", expr, resp.Result.Value())
		}
		if got.ValueBool() != want {
			t.Errorf("%q: got %v, want %v", expr, got.ValueBool(), want)
		}
	}
}
