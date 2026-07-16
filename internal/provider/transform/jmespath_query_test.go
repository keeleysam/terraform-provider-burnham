package transform

import (
	"encoding/json"
	"reflect"
	"testing"
)

// runJMESPath is the pure core of the jmespath_query function: it evaluates an
// expression against a value in the JSON value space (json.Number for numbers)
// and returns the matching result.

func booksData() map[string]interface{} {
	return map[string]interface{}{
		"books": []interface{}{
			map[string]interface{}{"title": "cheap", "price": json.Number("5")},
			map[string]interface{}{"title": "mid", "price": json.Number("10")},
			map[string]interface{}{"title": "dear", "price": json.Number("20")},
		},
	}
}

func TestRunJMESPath_NumericFilter(t *testing.T) {
	got, err := runJMESPath(booksData(), "books[?price < `10`].title")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []interface{}{"cheap"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestRunJMESPath_Arithmetic(t *testing.T) {
	data := map[string]interface{}{"a": json.Number("3"), "b": json.Number("4")}
	got, err := runJMESPath(data, "a + b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != float64(7) {
		t.Errorf("got %#v, want 7", got)
	}
}

func TestRunJMESPath_NumericFunctions(t *testing.T) {
	for _, tc := range []struct {
		expr string
		want float64
	}{
		{"max(books[].price)", 20},
		{"sum(books[].price)", 35},
		{"avg(books[].price)", 35.0 / 3.0},
	} {
		got, err := runJMESPath(booksData(), tc.expr)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.expr, err)
		}
		if got != tc.want {
			t.Errorf("%s: got %#v, want %v", tc.expr, got, tc.want)
		}
	}
}
