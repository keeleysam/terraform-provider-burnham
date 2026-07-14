package cel

import "testing"

func TestEncodeCallsAndMacros(t *testing.T) {
	cases := []struct {
		name string
		node any
		want string
	}{
		{
			"global call",
			m("call", map[string]any{
				"function": "inIpRange",
				"args":     []any{m("ident", "origin.ip"), []any{"203.0.113.24"}},
			}),
			`inIpRange(origin.ip, ["203.0.113.24"])`,
		},
		{
			"method call",
			m("call", map[string]any{
				"function": "startsWith",
				"target":   m("ident", "resource.name"),
				"args":     []any{"prod-"},
			}),
			`resource.name.startsWith("prod-")`,
		},
		{
			"method call no args",
			m("call", map[string]any{
				"function": "clientCertFingerprint",
				"target":   m("ident", "origin"),
			}),
			`origin.clientCertFingerprint()`,
		},
		{
			"has macro as global call",
			m("call", map[string]any{
				"function": "has",
				"args":     []any{m("ident", "device.vendors.acme")},
			}),
			`has(device.vendors.acme)`,
		},
		{
			"exists macro as method call with bound var",
			m("call", map[string]any{
				"function": "exists",
				"target":   m("ident", "user.groups"),
				"args": []any{
					m("ident", "g"),
					m("call", map[string]any{
						"function": "startsWith",
						"target":   m("ident", "g"),
						"args":     []any{"admin-"},
					}),
				},
			}),
			`user.groups.exists(g, g.startsWith("admin-"))`,
		},
		{
			"unknown function just works",
			m("call", map[string]any{
				"function": "someCustomFn",
				"args":     []any{m("ident", "a"), m("ident", "b")},
			}),
			`someCustomFn(a, b)`,
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
		{"missing function", m("call", map[string]any{"args": []any{}})},
		{"function not string", m("call", map[string]any{"function": 5})},
		{"args not list", m("call", map[string]any{"function": "f", "args": "nope"})},
		{"unknown call key", m("call", map[string]any{"function": "f", "bogus": 1})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Encode(tc.node); err == nil {
				t.Fatalf("Encode(%v) = nil error, want error", tc.node)
			}
		})
	}
}
