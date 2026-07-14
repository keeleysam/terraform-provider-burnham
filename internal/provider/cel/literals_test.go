package cel

import "testing"

func TestEncodeConstAndStructAndRaw(t *testing.T) {
	cases := []struct {
		name string
		node any
		want string
	}{
		{"const scalar string", m("const", "US"), `"US"`},
		{"const list literal", m("const", []any{"US", "FR"}), `["US", "FR"]`},
		{"const map literal sorted", m("const", map[string]any{"b": 2, "a": 1}), `{"a": 1, "b": 2}`},
		{"const typed double", m("const", map[string]any{"double_value": 5}), `5.0`},
		{"const typed uint", m("const", map[string]any{"uint64_value": 5}), `5u`},
		{"const typed bytes", m("const", map[string]any{"bytes_value": "hi"}), `b"\150\151"`},
		{"const_expr canonical typed", m("const_expr", map[string]any{"double_value": 2.5}), `2.5`},
		{
			"struct message construction",
			m("struct", map[string]any{
				"message_name": "google.protobuf.Timestamp",
				"fields":       map[string]any{"seconds": 1},
			}),
			`google.protobuf.Timestamp{seconds: 1}`,
		},
		{"raw passthrough", m("raw", `a > 1 && b in [1, 2]`), `a > 1 && b in [1, 2]`},
		{"raw with macro", m("raw", `x.exists(y, y > 0)`), `x.exists(y, y > 0)`},
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
