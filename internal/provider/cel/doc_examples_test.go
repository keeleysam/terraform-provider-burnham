package cel

import "testing"

// TestEncodeRealWorldExamples checks that celencode reproduces real CEL expressions drawn from official Kubernetes and GCP documentation.
// Each `node` is the HCL data tree; `want` is the CEL it must emit.
func TestEncodeRealWorldExamples(t *testing.T) {
	cases := []struct {
		name string
		node any
		want string
	}{
		{
			"acm region eq",
			m("==", []any{m("ident", "origin.region_code"), "GB"}),
			`origin.region_code == "GB"`,
		},
		{
			"acm region in list",
			m("in", []any{m("ident", "origin.region_code"), []any{"US", "FR", "JP"}}),
			`origin.region_code in ["US", "FR", "JP"]`,
		},
		{
			"acm inIpRange",
			m("call", map[string]any{"function": "inIpRange", "args": []any{m("ident", "origin.ip"), []any{"203.0.113.24"}}}),
			`inIpRange(origin.ip, ["203.0.113.24"])`,
		},
		{
			"acm enum eq",
			m("==", []any{m("ident", "device.os_type"), m("ident", "OsType.DESKTOP_MAC")}),
			`device.os_type == OsType.DESKTOP_MAC`,
		},
		{
			"acm vendor index",
			m("==", []any{m("ident", "device.vendors['some_vendor'].is_compliant_device"), true}),
			`device.vendors["some_vendor"].is_compliant_device == true`,
		},
		{
			"acm time method",
			m("==", []any{
				m("call", map[string]any{"target": m("ident", "request.time"), "function": "getHours", "args": []any{"America/Los_Angeles"}}),
				19,
			}),
			`request.time.getHours("America/Los_Angeles") == 19`,
		},
		{
			"acm enum list membership",
			m("in", []any{
				m("ident", "device.chrome.management_state"),
				[]any{m("ident", "ChromeManagementState.CHROME_MANAGEMENT_STATE_BROWSER_MANAGED"), m("ident", "ChromeManagementState.CHROME_MANAGEMENT_STATE_PROFILE_MANAGED")},
			}),
			`device.chrome.management_state in [ChromeManagementState.CHROME_MANAGEMENT_STATE_BROWSER_MANAGED, ChromeManagementState.CHROME_MANAGEMENT_STATE_PROFILE_MANAGED]`,
		},
		{
			"k8s startsWith",
			m("call", map[string]any{"target": m("ident", "self.health"), "function": "startsWith", "args": []any{"ok"}}),
			`self.health.startsWith("ok")`,
		},
		{
			"k8s has and arithmetic",
			m("&&", []any{
				m("call", map[string]any{"function": "has", "args": []any{m("ident", "self.expired")}}),
				m("<", []any{m("+", []any{m("ident", "self.created"), m("ident", "self.ttl")}), m("ident", "self.expired")}),
			}),
			`has(self.expired) && self.created + self.ttl < self.expired`,
		},
		{
			"k8s exists macro",
			m("call", map[string]any{
				"target": m("ident", "self.widgets"), "function": "exists",
				"args": []any{m("ident", "w"), m("&&", []any{m("==", []any{m("ident", "w.key"), "x"}), m("<", []any{m("ident", "w.foo"), 10})})},
			}),
			`self.widgets.exists(w, w.key == "x" && w.foo < 10)`,
		},
		{
			"k8s all with negated membership",
			m("call", map[string]any{
				"target": m("ident", "self.set1"), "function": "all",
				"args": []any{m("ident", "e"), m("!", m("in", []any{m("ident", "e"), m("ident", "self.set2")}))},
			}),
			`self.set1.all(e, !(e in self.set2))`,
		},
		{
			"k8s map sum double",
			m("==", []any{
				m("call", map[string]any{
					"target":   m("call", map[string]any{"target": m("ident", "items"), "function": "map", "args": []any{m("ident", "x"), m("ident", "x.weight")}}),
					"function": "sum",
				}),
				m("const", map[string]any{"double_value": 1}),
			}),
			`items.map(x, x.weight).sum() == 1.0`,
		},
		{
			"k8s ternary type check",
			m("cond", []any{
				m("==", []any{m("call", map[string]any{"function": "type", "args": []any{m("ident", "self")}}), m("ident", "string")}),
				m("==", []any{m("ident", "self"), "99%"}),
				m("==", []any{m("ident", "self"), 42}),
			}),
			`(type(self) == string) ? (self == "99%") : (self == 42)`,
		},
		{
			"k8s regex findAll chain",
			m("<", []any{
				m("call", map[string]any{
					"target": m("call", map[string]any{
						"target":   m("call", map[string]any{"target": "1, 2, 3, 4", "function": "findAll", "args": []any{"[0-9]+"}}),
						"function": "map", "args": []any{m("ident", "x"), m("call", map[string]any{"function": "int", "args": []any{m("ident", "x")}})},
					}),
					"function": "sum",
				}),
				100,
			}),
			`"1, 2, 3, 4".findAll("[0-9]+").map(x, int(x)).sum() < 100`,
		},
		{
			"k8s cidr containsIP",
			m("call", map[string]any{
				"target":   m("call", map[string]any{"function": "cidr", "args": []any{"192.168.0.0/24"}}),
				"function": "containsIP", "args": []any{m("call", map[string]any{"function": "ip", "args": []any{"192.168.0.1"}})},
			}),
			`cidr("192.168.0.0/24").containsIP(ip("192.168.0.1"))`,
		},
		{
			"k8s quantity compare",
			m("call", map[string]any{
				"target":   m("call", map[string]any{"function": "quantity", "args": []any{"150Mi"}}),
				"function": "isGreaterThan", "args": []any{m("call", map[string]any{"function": "quantity", "args": []any{"100Mi"}})},
			}),
			`quantity("150Mi").isGreaterThan(quantity("100Mi"))`,
		},
		{
			"cas subject_alt_names all",
			m("call", map[string]any{
				"target": m("ident", "subject_alt_names"), "function": "all",
				"args": []any{m("ident", "san"), m("&&", []any{
					m("==", []any{m("ident", "san.type"), m("ident", "DNS")}),
					m("call", map[string]any{"target": m("ident", "san.value"), "function": "endsWith", "args": []any{".test.com"}}),
				})},
			}),
			`subject_alt_names.all(san, san.type == DNS && san.value.endsWith(".test.com"))`,
		},
		{
			"cas size and oid list",
			m("&&", []any{
				m("==", []any{m("call", map[string]any{"target": m("ident", "subject_alt_names"), "function": "size"}), 1}),
				m("==", []any{m("ident", "subject_alt_names[0].oid"), []any{1, 2, 3, 5, 17}}),
			}),
			`subject_alt_names.size() == 1 && subject_alt_names[0].oid == [1, 2, 3, 5, 17]`,
		},
		{
			"k8s optional hasValue value select",
			m("&&", []any{
				m("call", map[string]any{"target": m("ident", "oldSelf"), "function": "hasValue"}),
				m("!=", []any{
					m("select_expr", map[string]any{"operand": m("call", map[string]any{"target": m("ident", "oldSelf"), "function": "value"}), "field": "foo"}),
					"foo",
				}),
			}),
			`oldSelf.hasValue() && oldSelf.value().foo != "foo"`,
		},
		{
			"k8s optional navigation orValue",
			m("call", map[string]any{"target": m("ident", "msg.?field"), "function": "orValue", "args": []any{0}}),
			`msg.?field.orValue(0)`,
		},
		{
			"k8s two-variable comprehension",
			m("call", map[string]any{
				"target": m("ident", "m"), "function": "all",
				"args": []any{m("ident", "k"), m("ident", "v"), m(">", []any{m("ident", "v"), 0})},
			}),
			`m.all(k, v, v > 0)`,
		},
		{
			"svc-ext path and header membership",
			m("&&", []any{
				m("call", map[string]any{"target": m("ident", "request.path"), "function": "endsWith", "args": []any{"/inventory"}}),
				m("in", []any{"Hello", m("ident", "request.headers")}),
			}),
			`request.path.endsWith("/inventory") && "Hello" in request.headers`,
		},
		{
			"k8s optional list literal",
			[]any{"a", m("optional", m("ident", "b"))},
			`["a", ?b]`,
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
