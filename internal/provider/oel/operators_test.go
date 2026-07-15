package oel

import "testing"

func TestEncodeOperators(t *testing.T) {
	uCity := m("ident", "user.city")
	uSalary := m("ident", "user.salary")
	uContractor := m("ident", "user.isContractor")
	a := m("ident", "user.a")
	b := m("ident", "user.b")
	c := m("ident", "user.c")

	cases := []struct {
		name string
		node any
		want string
	}{
		{"eq token", m("==", []any{uCity, "San Francisco"}), `user.city=="San Francisco"`},
		{"eq alias", m("eq", []any{uCity, "San Francisco"}), `user.city=="San Francisco"`},
		{"gt token", m(">", []any{uSalary, 1000000}), `user.salary>1000000`},
		{"ne alias empty string", m("ne", []any{m("ident", "user.employeeNumber"), ""}), `user.employeeNumber!=""`},
		{"not token", m("!", uContractor), `!user.isContractor`},
		{"not alias", m("not", uContractor), `!user.isContractor`},
		{"and token", m("and", []any{m(">", []any{uSalary, 1000000}), m("!", uContractor)}), `user.salary>1000000 AND !user.isContractor`},
		{"or alias three operands", m("or", []any{a, b, c}), `user.a OR user.b OR user.c`},
		{"single-operand and unwraps", m("and", []any{a}), `user.a`},
		{"additive fold", m("+", []any{m("ident", "user.firstName"), ".", m("ident", "user.lastName")}), `user.firstName + "." + user.lastName`},
		{"ternary token", m("?:", []any{m("==", []any{m("ident", "user.groupCode"), 123}), "Sales", "Other"}), `user.groupCode==123 ? "Sales" : "Other"`},
		{"ternary alias cond", m("cond", []any{m("==", []any{m("ident", "user.groupCode"), 123}), "Sales", "Other"}), `user.groupCode==123 ? "Sales" : "Other"`},
		{"precedence parens", m("or", []any{m("and", []any{a, b}), c}), `(user.a AND user.b) OR user.c`},
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
	a := m("ident", "user.a")
	cases := []struct {
		name string
		node any
	}{
		{"eq needs two", m("==", []any{a})},
		{"eq rejects three", m("==", []any{a, a, a})},
		{"ternary needs three", m("cond", []any{a, a})},
		{"additive needs two", m("+", []any{a})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Encode(tc.node); err == nil {
				t.Fatalf("Encode(%v) = nil error, want arity error", tc.node)
			}
		})
	}
}
