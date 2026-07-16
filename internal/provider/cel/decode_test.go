package cel

import "testing"

// The strongest correctness check: decode a CEL string into each notation, then
// re-encode, and confirm every notation reproduces the canonical CEL.
func TestRoundTrip(t *testing.T) {
	exprs := []string{
		`origin.region_code == "GB"`,
		`device.os_type == OsType.DESKTOP_MAC`,
		`origin.region_code in ["US", "FR", "JP"]`,
		`inIpRange(origin.ip, ["203.0.113.24"])`,
		`request.time.getHours("America/Los_Angeles") == 19`,
		`device.vendors["some_vendor"].is_compliant_device == true`,
		`self.health.startsWith("ok")`,
		`has(self.expired) && self.created + self.ttl < self.expired`,
		`self.widgets.exists(w, w.key == "x" && w.foo < 10)`,
		`self.set1.all(e, !(e in self.set2))`,
		`items.map(x, x.weight).sum() == 1.0`,
		`cidr("192.168.0.0/24").containsIP(ip("192.168.0.1"))`,
		`subject_alt_names.all(san, san.type == DNS && san.value.endsWith(".test.com"))`,
		`subject_alt_names.size() == 1 && subject_alt_names[0].oid == [1, 2, 3, 5, 17]`,
		`oldSelf.hasValue() && oldSelf.value().foo != "foo"`,
		`msg.?field.orValue(0)`,
		`m.all(k, v, v > 0)`,
		`request.path.endsWith("/inventory") && "Hello" in request.headers`,
		`["a", ?b]`,
		`a && b && c || d`,
		`-x < 5`,
		// unary operator applied to a list literal (operand must not be spread into the operand sequence)
		`-[1, 2]`,
		`![a, b]`,
		`type(self) == string ? self == "99%" : self == 42`,
		`quantity("150Mi").isGreaterThan(quantity("100Mi"))`,
		`{"k": v, "j": w}`,
		// precedence / associativity
		`a - b - c`,
		`a / b / c`,
		`!!x`,
		`a > 0 ? b : c > 0 ? d : e`,
		`-9223372036854775807 < 0`,
		// aggregates and construction
		`Msg{field: 1}`,
		`pkg.Type{a: 1, b: x}`,
		`{1: "a", 2: "b"}`,
		`[[1, 2], [3]]`,
		// macros: 3-arg map, transforms, nested with shadow
		`l.map(x, x > 0, x * 2)`,
		`m.transformList(k, v, v + 1)`,
		`m.transformMapEntry(k, v, {k: v})`,
		`x.exists(y, x.all(z, z == y))`,
		`x.exists(y, has(y.z))`,
		// optional entries and chained optional navigation
		`Msg{?field: v}`,
		`{?"k": v}`,
		`msg.?a.?b.orValue(0)`,
		// strings / dyn
		`x == "a\nb\t\"c\""`,
		`dyn(x) == 1`,
	}
	modes := []string{"canonical", "standard", "aliased"}
	for _, ex := range exprs {
		want, err := Format(ex)
		if err != nil {
			t.Fatalf("Format(%q) setup error: %v", ex, err)
		}
		for _, mode := range modes {
			node, err := Decode(ex, mode)
			if err != nil {
				t.Errorf("Decode(%q, %s) error: %v", ex, mode, err)
				continue
			}
			got, err := Encode(node)
			if err != nil {
				t.Errorf("Encode(Decode(%q, %s)) error: %v", ex, mode, err)
				continue
			}
			if got != want {
				t.Errorf("round-trip mismatch (%s)\n  in:   %q\n  want: %q\n  got:  %q", mode, ex, want, got)
			}
		}
	}
}
