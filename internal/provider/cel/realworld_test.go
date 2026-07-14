package cel

import "testing"

// Verbatim CEL expressions drawn from official Kubernetes and GCP documentation.
// celvalidate must accept every one, which exercises the optional-types and
// two-variable-comprehension libraries plus the open function space.
func TestValidateRealWorldExamples(t *testing.T) {
	examples := []string{
		// Kubernetes CEL reference / CRD validation / ValidatingAdmissionPolicy
		`self.minReplicas <= self.replicas && self.replicas <= self.maxReplicas`,
		`self.widgets.exists(w, w.key == 'x' && w.foo < 10)`,
		`self.details.all(key, self.details[key].matches('^[a-zA-Z]*$'))`,
		`items.map(x, x.weight).sum() == 1.0`,
		`"1, 2, 3, 4".findAll('[0-9]+').map(x, int(x)).sum() < 100`,
		`url('https://example.com:80/').getHost()`,
		`cidr('192.168.0.0/24').containsIP(ip('192.168.0.1'))`,
		`quantity("150Mi").isGreaterThan(quantity("100Mi"))`,
		`isSemver('1.0.0')`,
		`authorizer.group('').resource('pods').namespace('default').check('create').allowed()`,
		`has(object.metadata.labels) && 'example.com/environment' in object.metadata.labels`,
		// Kubernetes optional types
		`oldSelf.hasValue() && oldSelf.value().foo != "foo"`,
		`oldSelf.optMap(o, o.size()).orValue(0) < 4 || self.size() >= 4`,
		`msg.?field.orValue(0) == 1`,
		`optional.of(x).orValue(y) == 1`,
		// Kubernetes two-variable comprehensions
		`m.all(k, v, v > 0)`,
		`l.transformList(i, v, v + 1) == [2]`,
		// GCP Certificate Authority Service
		`subject.common_name == "google.com" && (subject.country_code == "US" || subject.country_code == "IR")`,
		`subject_alt_names.all(san, san.type == DNS && san.value.endsWith(".test.com"))`,
		`subject_alt_names.size() == 1 && subject_alt_names[0].oid == [1, 2, 3, 5, 17]`,
		`api.getAttribute('privateca.googleapis.com/template', '') == 'my-project-pki/-/leaf-server-tls'`,
		// GCP Service Extensions matcher
		`request.host.endsWith('.example.com')`,
		`request.path.endsWith("/inventory") && "Hello" in request.headers`,
	}
	for _, ex := range examples {
		if _, err := Format(ex); err != nil {
			t.Errorf("Format(%q) rejected a documented real-world example: %v", ex, err)
		}
	}
}

func TestEncodeOptionalTypes(t *testing.T) {
	cases := []struct {
		name string
		node any
		want string
	}{
		{"optional select via ident path", m("ident", "msg.?field"), `msg.?field`},
		{"optional index via ident path", m("ident", "m[?k]"), `m[?k]`},
		{
			"optional list element",
			[]any{m("ident", "a"), m("optional", m("ident", "b"))},
			`[a, ?b]`,
		},
		{
			"optional map entry via struct_expr",
			m("struct_expr", map[string]any{
				"entries": []any{map[string]any{
					"map_key": m("ident", "k"), "value": m("ident", "v"), "optional_entry": true,
				}},
			}),
			`{?k: v}`,
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

func TestEncodeTwoVariableComprehension(t *testing.T) {
	node := m("call", map[string]any{
		"target":   m("ident", "m"),
		"function": "all",
		"args": []any{
			m("ident", "k"), m("ident", "v"),
			m(">", []any{m("ident", "v"), 0}),
		},
	})
	got, err := Encode(node)
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}
	if got != `m.all(k, v, v > 0)` {
		t.Fatalf("Encode = %q, want %q", got, `m.all(k, v, v > 0)`)
	}
}

func TestIdentReferenceGuard(t *testing.T) {
	// These are reference paths and must be accepted.
	for _, ok := range []string{"device.os_type", "OsType.DESKTOP_MAC", "device.vendors['acme'].id", "list[0]", "msg.?field"} {
		if _, err := Encode(m("ident", ok)); err != nil {
			t.Errorf("ident %q should be accepted, got %v", ok, err)
		}
	}
	// These are full expressions, not references, and must be rejected (so we never
	// silently emit an expression fed through `ident`).
	for _, bad := range []string{"a == b", "f(x)", "1 + 2", `"literal"`, "a && b"} {
		if _, err := Encode(m("ident", bad)); err == nil {
			t.Errorf("ident %q should be rejected as not a reference path", bad)
		}
	}
}
