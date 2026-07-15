package oel

import "testing"

// TestEncodeExtended covers the constructs beyond the classic namespaced subset
// (method calls, the Identity Engine dialect, bare functions, field access,
// indexing, projection, Elvis, matches, and map literals). These build the
// parser's own node types, so the output is canonical and is parsed back like
// everything else. Expected strings are the canonical (double-quoted) forms.
func TestEncodeExtended(t *testing.T) {
	cases := []struct {
		name string
		node any
		want string
	}{
		{
			"classic path method call getInternalProperty",
			m("call", map[string]any{"target": m("ident", "user"), "method": "getInternalProperty", "args": []any{"status"}}),
			`user.getInternalProperty("status")`,
		},
		{
			"IE method dialect toUpperCase no args",
			m("call", map[string]any{"target": m("ident", "user.profile.firstName"), "method": "toUpperCase"}),
			`user.profile.firstName.toUpperCase()`,
		},
		{
			"chained method calls",
			m("call", map[string]any{
				"target": m("call", map[string]any{"target": "USA", "method": "parseCountryCode"}),
				"method": "toAlpha2",
			}),
			`"USA".parseCountryCode().toAlpha2()`,
		},
		{
			"bare function call",
			m("call", map[string]any{"function": "substringBefore", "args": []any{m("ident", "user.email"), "@"}}),
			`substringBefore(user.email, "@")`,
		},
		{
			"field access on function-call receiver",
			m("select", map[string]any{
				"operand": m("call", map[string]any{"function": "getManagerUser", "args": []any{"active_directory"}}),
				"field":   "firstName",
			}),
			`getManagerUser("active_directory").firstName`,
		},
		{
			"index access",
			m("index", map[string]any{"base": m("ident", "user.arrayProperty"), "index": 0}),
			`user.arrayProperty[0]`,
		},
		{
			"elvis operator",
			m("elvis", []any{
				m("call", map[string]any{"class": "Groups", "method": "startsWith", "args": []any{"OKTA", "TEST", 100}}),
				[]any{},
			}),
			`Groups.startsWith("OKTA", "TEST", 100) ?: {}`,
		},
		{
			"matches operator",
			m("matches", []any{m("ident", "user.title"), "(?i)engineer"}),
			`user.title matches "(?i)engineer"`,
		},
		{
			"isMemberOf object form",
			m("call", map[string]any{
				"target": m("ident", "user"),
				"method": "isMemberOf",
				"args": []any{
					m("map", []any{
						map[string]any{"key": "group.profile.name", "value": "West Coast Users"},
						map[string]any{"key": "operator", "value": "EXACT"},
					}),
				},
			}),
			`user.isMemberOf({"group.profile.name": "West Coast Users", "operator": "EXACT"})`,
		},
		{
			"getGroups with projection",
			m("project", map[string]any{
				"base": m("call", map[string]any{
					"target": m("ident", "user"),
					"method": "getGroups",
					"args":   []any{m("map", []any{map[string]any{"key": "group.profile.name", "value": "Everyone"}})},
				}),
				"expr": m("ident", "profile.name"),
			}),
			`user.getGroups({"group.profile.name": "Everyone"}).![profile.name]`,
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
		})
	}
}
