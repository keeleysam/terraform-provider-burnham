package oel

import (
	"testing"

	okta "github.com/keeleysam/okta-expression-parser"
)

// mustParse asserts that s is valid Okta EL, guarding against emitting garbage.
func mustParse(t *testing.T, s string) {
	t.Helper()
	if _, err := okta.New().Parse(s); err != nil {
		t.Fatalf("emitted OEL is invalid: %q: %v", s, err)
	}
}

func TestEncodeLeaves(t *testing.T) {
	cases := []struct {
		name string
		node any
		want string
	}{
		{"string literal", "San Francisco", `"San Francisco"`},
		{"int literal", 123, `123`},
		{"bool literal true", true, `true`},
		{"bool literal false", false, `false`},
		{"null literal", nil, `null`},
		{"reference user root", m("ident", "user.city"), `user.city`},
		{"reference non-user root", m("ident", "appuser.email"), `appuser.email`},
		{"array literal", []any{10, 20, 30}, `{10, 20, 30}`},
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
