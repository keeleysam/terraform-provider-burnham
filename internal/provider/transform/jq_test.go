package transform

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

// runJQ is the pure core of the jq function: it runs a program against a value
// in the JSON value space (json.Number for numbers) and returns the output
// stream as a slice, also in the JSON value space.

func TestRunJQ_Identity(t *testing.T) {
	got, err := runJQ(context.Background(), map[string]interface{}{"a": json.Number("1")}, ".", nil)
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
	got, err := runJQ(context.Background(), input, ".user.name", nil)
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
	got, err := runJQ(context.Background(), input, ".[]", nil)
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
	got, err := runJQ(context.Background(), input, ".[] | select(. > 5)", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %#v", got)
	}
}

func TestRunJQ_Arithmetic(t *testing.T) {
	got, err := runJQ(context.Background(), json.Number("20"), ". + 22", nil)
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
	got, err := runJQ(context.Background(), json.Number("1"), ". + $limit", vars)
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
	got, err := runJQ(context.Background(), json.Number(big), ".", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []interface{}{json.Number(big)}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestRunJQ_InvalidProgram(t *testing.T) {
	_, err := runJQ(context.Background(), nil, ".[", nil)
	if err == nil {
		t.Fatal("expected error for invalid program, got nil")
	}
}

func TestRunJQ_RuntimeError(t *testing.T) {
	// Adding a string to a number is a runtime type error in jq.
	_, err := runJQ(context.Background(), map[string]interface{}{"a": "x"}, ".a + 1", nil)
	if err == nil {
		t.Fatal("expected runtime error, got nil")
	}
}

func TestRunJQ_NowIsAllowed(t *testing.T) {
	// now is permitted (nondeterministic, documented). Assert on its type so
	// the test itself stays deterministic.
	got, err := runJQ(context.Background(), nil, "now | type", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []interface{}{"number"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestRunJQ_DeeplyNestedResultBounded(t *testing.T) {
	/*
		A jq program can build a result nested far deeper than any real config, and the output conversion path must bound its recursion (like the input path) and return an error rather than overflowing the goroutine stack.

		Depth 2000 is deliberate, not arbitrary: it exceeds transformMaxDepth (1024) so the bound must trip, yet it is shallow enough that the UNFIXED code (no output-depth guard) returns a nil error rather than crashing. That is exactly the red/green boundary a regression test needs: fixed code returns the depth error, unfixed code returns nil. Reproducing the real crash depth (~2,000,000) is not usable here, because against the unfixed code that overflow aborts the test process itself, so nothing could be asserted.
	*/
	_, err := runJQ(context.Background(), json.Number("0"), "reduce range(2000) as $i (0; [.])", nil)
	if err == nil {
		t.Fatal("expected error for result nested beyond the maximum depth, got nil")
	}
	if !strings.Contains(err.Error(), "nesting depth") {
		t.Fatalf("expected the depth-bound error, got a different error: %v", err)
	}
}

func TestRunJQ_InfiniteProgramTimesOut(t *testing.T) {
	// jq is Turing-complete, so a non-terminating program must be bounded by the request context rather than hanging the plan forever.
	// A short deadline must produce an error promptly. The call runs in a goroutine with a watchdog so a regression (an unbounded run) fails the test instead of hanging it.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := runJQ(ctx, nil, "def f: f; f", nil)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error for non-terminating program, got nil")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("runJQ hung on a non-terminating program")
	}
}

func TestRunJQ_NonFiniteRejected(t *testing.T) {
	// jq's infinite/nan builtins emit non-finite numbers that a Terraform number cannot represent.
	// Both must surface a clear "non-finite" error at conversion, not a silent +Inf value or a misleading "number has no digits" parse error.
	for _, prog := range []string{"infinite", "-infinite", "nan"} {
		results, err := runJQ(context.Background(), nil, prog, nil)
		if err != nil {
			t.Fatalf("jq %q: unexpected runJQ error: %v", prog, err)
		}
		_, err = jsonToTerraform(results)
		if err == nil {
			t.Fatalf("jq %q: expected a non-finite number to be rejected, got nil error", prog)
		}
		if !strings.Contains(err.Error(), "non-finite") {
			t.Errorf("jq %q: expected a clear non-finite error, got %v", prog, err)
		}
	}
}

func TestRunJQ_EnvDoesNotLeakHostEnv(t *testing.T) {
	// env does not expose the host process environment; it is an empty object.
	t.Setenv("BURNHAM_JQ_SECRET", "leaked")
	got, err := runJQ(context.Background(), nil, "env.BURNHAM_JQ_SECRET", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []interface{}{nil}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("env leaked host environment: got %#v, want %#v", got, want)
	}
}
