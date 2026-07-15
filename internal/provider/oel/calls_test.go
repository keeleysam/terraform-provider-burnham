package oel

import "testing"

func TestEncodeClassCalls(t *testing.T) {
	cases := []struct {
		name string
		node any
		want string
	}{
		{
			"class call two args",
			m("call", map[string]any{"class": "String", "method": "stringContains",
				"args": []any{m("ident", "user.department"), "Sales"}}),
			`String.stringContains(user.department, "Sales")`,
		},
		{
			"class call one arg",
			m("call", map[string]any{"class": "String", "method": "toUpperCase", "args": []any{"This"}}),
			`String.toUpperCase("This")`,
		},
		{
			"class call nested",
			m("call", map[string]any{"class": "String", "method": "startsWith",
				"args": []any{m("ident", "user.firstName"),
					m("call", map[string]any{"class": "String", "method": "toLowerCase", "args": []any{"bob"}})}}),
			`String.startsWith(user.firstName, String.toLowerCase("bob"))`,
		},
		{
			"class call array arg",
			m("call", map[string]any{"class": "Arrays", "method": "contains",
				"args": []any{[]any{10, 20, 30}, 10}}),
			`Arrays.contains({10, 20, 30}, 10)`,
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

func TestEncodeGroupBuiltins(t *testing.T) {
	cases := []struct {
		name string
		node any
		want string
	}{
		{
			"is member of group name",
			m("call", map[string]any{"function": "isMemberOfGroupName", "args": []any{"group1"}}),
			`isMemberOfGroupName("group1")`,
		},
		{
			"is member of any group multi",
			m("call", map[string]any{"function": "isMemberOfAnyGroup", "args": []any{"a", "b", "c"}}),
			`isMemberOfAnyGroup("a", "b", "c")`,
		},
		{
			"is member of group name regex",
			m("call", map[string]any{"function": "isMemberOfGroupNameRegex", "args": []any{"/.*admin.*/"}}),
			`isMemberOfGroupNameRegex("/.*admin.*/")`,
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

func TestEncodeCallErrors(t *testing.T) {
	cases := []struct {
		name string
		node any
	}{
		{"missing method", m("call", map[string]any{"class": "String"})},
		{"class and function together", m("call", map[string]any{"class": "String", "method": "x", "function": "isMemberOfGroup"})},
		{"target and function together", m("call", map[string]any{"target": m("ident", "user"), "method": "x", "function": "isMemberOfGroup"})},
		{"neither class nor function nor target", m("call", map[string]any{"args": []any{}})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Encode(tc.node); err == nil {
				t.Fatalf("Encode(%v) = nil error, want error", tc.node)
			}
		})
	}
}
