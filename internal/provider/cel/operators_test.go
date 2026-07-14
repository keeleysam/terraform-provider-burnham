package cel

import "testing"

func TestEncodeOperators(t *testing.T) {
	a := m("ident", "a")
	b := m("ident", "b")
	c := m("ident", "c")

	cases := []struct {
		name string
		node any
		want string
	}{
		{"eq token", m("==", []any{a, "b"}), `a == "b"`},
		{"eq alias", m("eq", []any{a, "b"}), `a == "b"`},
		{"ne alias", m("ne", []any{a, b}), `a != b`},
		{"lt alias", m("lt", []any{a, 5}), `a < 5`},
		{"and token variadic fold", m("&&", []any{a, b, c}), `a && b && c`},
		{"and alias variadic fold", m("and", []any{a, b, c}), `a && b && c`},
		{"or alias", m("or", []any{a, b}), `a || b`},
		{"not token unary single operand", m("!", a), `!a`},
		{"not alias", m("not", a), `!a`},
		{"neg alias", m("neg", a), `-a`},
		{"minus unary by arity", m("-", []any{a}), `-a`},
		{"minus binary by arity", m("-", []any{a, b}), `a - b`},
		{"in token", m("in", []any{a, []any{"US", "CA"}}), `a in ["US", "CA"]`},
		{"cond alias ternary", m("cond", []any{a, b, c}), `a ? b : c`},
		{"ternary token", m("?:", []any{a, b, c}), `a ? b : c`},
		{"precedence parens", m("&&", []any{m("||", []any{a, b}), c}), `(a || b) && c`},
		{"single-element and returns element", m("and", []any{a}), `a`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Encode(tc.node)
			if err != nil {
				t.Fatalf("Encode error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("Encode = %q, want %q", got, tc.want)
			}
			mustParse(t, got)
		})
	}
}

func TestEncodeOperatorArityErrors(t *testing.T) {
	a := m("ident", "a")
	cases := []struct {
		name string
		node any
	}{
		{"eq needs two", m("==", []any{a})},
		{"eq rejects three", m("==", []any{a, a, a})},
		{"cond needs three", m("cond", []any{a, a})},
		{"not rejects two", m("!", []any{a, a})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Encode(tc.node); err == nil {
				t.Fatalf("Encode(%v) = nil error, want arity error", tc.node)
			}
		})
	}
}
