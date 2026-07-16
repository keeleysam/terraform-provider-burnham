package cel

import (
	"testing"

	celgo "github.com/google/cel-go/cel"
)

// m is a tiny helper for building single-key node objects in tests.
func m(k string, v any) map[string]any { return map[string]any{k: v} }

// mustParse asserts that s is syntactically valid CEL (defense against emitting garbage).
func mustParse(t *testing.T, s string) {
	t.Helper()
	env, _ := celgo.NewEnv()
	if _, iss := env.Parse(s); iss != nil && iss.Err() != nil {
		t.Fatalf("emitted CEL is invalid: %q: %v", s, iss.Err())
	}
}

// TestEncodeUnaryMacroMinimalParens guards against redundant parentheses around a
// receiver-macro call when it is the operand of a unary operator. A member call like
// l.map(x, c) binds tighter than unary minus, so -l.map(x, c) needs no parentheses.
// The pretty path already emits the minimal form; this pins the non-pretty path to it
// so the two canonical paths stay equivalent.
func TestEncodeUnaryMacroMinimalParens(t *testing.T) {
	node := m("call", map[string]any{
		"function": "-_",
		"args": []any{
			m("call", map[string]any{
				"target":   m("ident", "l"),
				"function": "map",
				"args":     []any{m("ident", "x"), m("ident", "c")},
			}),
		},
	})
	want := "-l.map(x, c)"

	got, err := Encode(node)
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}
	if got != want {
		t.Fatalf("Encode = %q, want %q", got, want)
	}
	mustParse(t, got)

	gotP, err := Encode(node, Pretty())
	if err != nil {
		t.Fatalf("Encode(Pretty) error: %v", err)
	}
	if gotP != want {
		t.Fatalf("Encode(Pretty) = %q, want %q (two paths must agree)", gotP, want)
	}
}

func TestEncodeLeaves(t *testing.T) {
	cases := []struct {
		name string
		node any
		want string
	}{
		{"string literal", "US", `"US"`},
		{"int literal", 19, `19`},
		{"bool literal", true, `true`},
		{"null literal", nil, `null`},
		{"reference", m("ident", "device.os_type"), `device.os_type`},
		{"reference with index", m("ident", "m['k']"), `m["k"]`},
		{"bare list of string literals", []any{"US", "FR"}, `["US", "FR"]`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Encode(tc.node)
			if err != nil {
				t.Fatalf("Encode(%v) error: %v", tc.node, err)
			}
			if got != tc.want {
				t.Fatalf("Encode(%v) = %q, want %q", tc.node, got, tc.want)
			}
			mustParse(t, got)
		})
	}
}
