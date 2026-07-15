package oel

import "testing"

// TestEncodeRealWorldExamples assembles node trees for real Okta group-rule and
// profile-mapping expressions taken from Okta's docs, and checks each encodes to
// the expected canonical OEL (and re-parses).
func TestEncodeRealWorldExamples(t *testing.T) {
	cases := []struct {
		name string
		node any
		want string
	}{
		{
			// okta_group_rule: first name starts with a case-normalized literal.
			"group rule nested class call",
			m("call", map[string]any{"class": "String", "method": "startsWith",
				"args": []any{
					m("ident", "user.firstName"),
					m("call", map[string]any{"class": "String", "method": "toLowerCase", "args": []any{"bob"}}),
				}}),
			`String.startsWith(user.firstName, String.toLowerCase("bob"))`,
		},
		{
			// High-salary, non-contractor users.
			"boolean chain with negation",
			m("and", []any{
				m(">", []any{m("ident", "user.salary"), 1000000}),
				m("!", m("ident", "user.isContractor")),
			}),
			`user.salary>1000000 AND !user.isContractor`,
		},
		{
			"department contains",
			m("call", map[string]any{"class": "String", "method": "stringContains",
				"args": []any{m("ident", "user.department"), "Sales"}}),
			`String.stringContains(user.department, "Sales")`,
		},
		{
			// Profile-mapping ternary.
			"ternary value selection",
			m("cond", []any{
				m("==", []any{m("ident", "user.groupCode"), 123}),
				"Sales", "Other",
			}),
			`user.groupCode==123 ? "Sales" : "Other"`,
		},
		{
			"is member of any group by id",
			m("call", map[string]any{"function": "isMemberOfAnyGroup", "args": []any{"00gb4o8b4kFEKqzMI0h7"}}),
			`isMemberOfAnyGroup("00gb4o8b4kFEKqzMI0h7")`,
		},
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
