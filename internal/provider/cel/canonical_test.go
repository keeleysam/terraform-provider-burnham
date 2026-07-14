package cel

import "testing"

func TestEncodeCanonicalPathA(t *testing.T) {
	cases := []struct {
		name string
		node any
		want string
	}{
		{"ident_expr", m("ident_expr", map[string]any{"name": "device"}), `device`},
		{
			"select_expr",
			m("select_expr", map[string]any{
				"operand": m("ident_expr", map[string]any{"name": "device"}),
				"field":   "os_type",
			}),
			`device.os_type`,
		},
		{
			"select_expr test_only is has()",
			m("select_expr", map[string]any{
				"operand":   m("ident_expr", map[string]any{"name": "msg"}),
				"field":     "field",
				"test_only": true,
			}),
			`has(msg.field)`,
		},
		{
			"call_expr operator",
			m("call_expr", map[string]any{
				"function": "_==_",
				"args": []any{
					m("select_expr", map[string]any{"operand": m("ident_expr", map[string]any{"name": "device"}), "field": "os_type"}),
					m("ident_expr", map[string]any{"name": "OsType"}),
				},
			}),
			`device.os_type == OsType`,
		},
		{
			"list_expr",
			m("list_expr", map[string]any{"elements": []any{"US", "CA"}}),
			`["US", "CA"]`,
		},
		{
			"struct_expr message with field_key",
			m("struct_expr", map[string]any{
				"message_name": "Msg",
				"entries":      []any{map[string]any{"field_key": "a", "value": 1}},
			}),
			`Msg{a: 1}`,
		},
		{
			"struct_expr map with map_key",
			m("struct_expr", map[string]any{
				"entries": []any{map[string]any{"map_key": "k", "value": m("ident", "v")}},
			}),
			`{"k": v}`,
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

// A Path A canonical subtree nested inside an otherwise Path B tree.
func TestEncodeMixedNotations(t *testing.T) {
	node := m("&&", []any{
		m("==", []any{m("ident", "device.os_type"), m("ident", "OsType.DESKTOP_MAC")}),
		// canonical subtree:
		m("call_expr", map[string]any{
			"function": "_>_",
			"args":     []any{m("ident_expr", map[string]any{"name": "size"}), 0},
		}),
	})
	got, err := Encode(node)
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}
	want := `device.os_type == OsType.DESKTOP_MAC && size > 0`
	if got != want {
		t.Fatalf("Encode = %q, want %q", got, want)
	}
	mustParse(t, got)
}

func TestEncodeComprehensionExprRejected(t *testing.T) {
	// A raw comprehension cannot be unparsed; authors must use the macro call form.
	node := m("comprehension_expr", map[string]any{"iter_var": "x"})
	if _, err := Encode(node); err == nil {
		t.Fatalf("expected comprehension_expr to be rejected with guidance")
	}
}
