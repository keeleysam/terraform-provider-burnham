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
