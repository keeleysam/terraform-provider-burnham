package transform

import (
	"encoding/json"
	"reflect"
	"testing"
)

// runJQ is the pure core of the jq function: it runs a program against a value
// in the JSON value space (json.Number for numbers) and returns the output
// stream as a slice, also in the JSON value space.

func TestRunJQ_Identity(t *testing.T) {
	got, err := runJQ(map[string]interface{}{"a": json.Number("1")}, ".", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []interface{}{map[string]interface{}{"a": json.Number("1")}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestRunJQ_FieldExtract(t *testing.T) {
	input := map[string]interface{}{"user": map[string]interface{}{"name": "alice"}}
	got, err := runJQ(input, ".user.name", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []interface{}{"alice"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestRunJQ_StreamBecomesList(t *testing.T) {
	input := []interface{}{json.Number("1"), json.Number("2"), json.Number("3")}
	got, err := runJQ(input, ".[]", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []interface{}{json.Number("1"), json.Number("2"), json.Number("3")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestRunJQ_EmptyStream(t *testing.T) {
	input := []interface{}{json.Number("1"), json.Number("2")}
	got, err := runJQ(input, ".[] | select(. > 5)", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %#v", got)
	}
}

func TestRunJQ_Arithmetic(t *testing.T) {
	got, err := runJQ(json.Number("20"), ". + 22", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []interface{}{json.Number("42")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestRunJQ_Vars(t *testing.T) {
	vars := map[string]interface{}{"limit": json.Number("5")}
	got, err := runJQ(json.Number("1"), ". + $limit", vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []interface{}{json.Number("6")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestRunJQ_BigIntegerPreserved(t *testing.T) {
	// 2^60 + 1 is beyond float64 integer precision; it must round-trip exactly.
	big := "1152921504606846977"
	got, err := runJQ(json.Number(big), ".", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []interface{}{json.Number(big)}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestRunJQ_InvalidProgram(t *testing.T) {
	_, err := runJQ(nil, ".[", nil)
	if err == nil {
		t.Fatal("expected error for invalid program, got nil")
	}
}

func TestRunJQ_RuntimeError(t *testing.T) {
	// Adding a string to a number is a runtime type error in jq.
	_, err := runJQ(map[string]interface{}{"a": "x"}, ".a + 1", nil)
	if err == nil {
		t.Fatal("expected runtime error, got nil")
	}
}

func TestRunJQ_NowIsAllowed(t *testing.T) {
	// now is permitted (nondeterministic — documented). Assert on its type so
	// the test itself stays deterministic.
	got, err := runJQ(nil, "now | type", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []interface{}{"number"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestRunJQ_EnvDoesNotLeakHostEnv(t *testing.T) {
	// env does not expose the host process environment; it is an empty object.
	t.Setenv("BURNHAM_JQ_SECRET", "leaked")
	got, err := runJQ(nil, "env.BURNHAM_JQ_SECRET", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []interface{}{nil}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("env leaked host environment: got %#v, want %#v", got, want)
	}
}
