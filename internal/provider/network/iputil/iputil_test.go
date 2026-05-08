package iputil

import (
	"testing"
)

// fatal is a small helper for (value, error) pairs where error should be fatal.
// Call as: v, err := F(); v = fatal(t, v, err)
// Or inline-style below where convenient.
func fatalOnErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
func wantErr(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- netipx-backed wrappers (smoke level) -----------------------------------

func TestMergeCIDRs(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{"siblings merge", []string{"10.0.0.0/24", "10.0.1.0/24"}, []string{"10.0.0.0/23"}},
		{"redundant dropped", []string{"10.0.0.0/24", "10.0.0.0/25"}, []string{"10.0.0.0/24"}},
		{"no merge possible", []string{"10.0.0.0/24", "10.0.2.0/24"}, []string{"10.0.0.0/24", "10.0.2.0/24"}},
		{"empty input", []string{}, []string{}},
		{"single", []string{"192.168.0.0/16"}, []string{"192.168.0.0/16"}},
		{"ipv6 siblings", []string{"2001:db8::/33", "2001:db8:8000::/33"}, []string{"2001:db8::/32"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := MergeCIDRs(tc.input)
			fatalOnErr(t, err)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("[%d] got %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestSubtractCIDRs(t *testing.T) {
	result, err := SubtractCIDRs([]string{"10.0.0.0/8"}, []string{"10.1.0.0/16"})
	fatalOnErr(t, err)
	if len(result) != 8 {
		t.Errorf("expected 8 CIDRs, got %d: %v", len(result), result)
	}

	result, err = SubtractCIDRs([]string{"10.0.0.0/24"}, []string{"10.0.0.0/24"})
	fatalOnErr(t, err)
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}

	result, err = SubtractCIDRs([]string{"10.0.0.0/24"}, []string{"192.168.0.0/16"})
	fatalOnErr(t, err)
	if len(result) != 1 || result[0] != "10.0.0.0/24" {
		t.Errorf("expected unchanged, got %v", result)
	}
}

func TestIntersectCIDRs(t *testing.T) {
	got, err := IntersectCIDRs([]string{"10.0.0.0/8"}, []string{"10.1.0.0/16"})
	fatalOnErr(t, err)
	if len(got) != 1 || got[0] != "10.1.0.0/16" {
		t.Errorf("got %v", got)
	}

	got, err = IntersectCIDRs([]string{"10.0.0.0/8"}, []string{"192.168.0.0/16"})
	fatalOnErr(t, err)
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

func TestRangeToCIDRs(t *testing.T) {
	got, err := RangeToCIDRs("10.0.0.1", "10.0.0.6")
	fatalOnErr(t, err)
	want := []string{"10.0.0.1/32", "10.0.0.2/31", "10.0.0.4/31", "10.0.0.6/32"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("[%d] got %q, want %q", i, got[i], want[i])
		}
	}

	got, err = RangeToCIDRs("10.0.0.5", "10.0.0.5")
	fatalOnErr(t, err)
	if len(got) != 1 || got[0] != "10.0.0.5/32" {
		t.Errorf("single IP: got %v", got)
	}

	_, err = RangeToCIDRs("10.0.0.6", "10.0.0.1")
	wantErr(t, err)
	_, err = RangeToCIDRs("10.0.0.1", "2001:db8::1")
	wantErr(t, err)
}

func TestCIDRsOverlapAny(t *testing.T) {
	got, err := CIDRsOverlapAny([]string{"10.0.0.0/8"}, []string{"10.1.0.0/16"})
	fatalOnErr(t, err)
	if !got {
		t.Error("expected true")
	}

	got, err = CIDRsOverlapAny([]string{"10.0.0.0/8"}, []string{"192.168.0.0/16"})
	fatalOnErr(t, err)
	if got {
		t.Error("expected false")
	}
}

// ---- IPAdd ------------------------------------------------------------------

func TestIPAdd(t *testing.T) {
	tests := []struct {
		ip   string
		n    int64
		want string
	}{
		{"10.0.0.0", 1, "10.0.0.1"},
		{"10.0.0.5", -3, "10.0.0.2"},
		{"10.0.0.255", 1, "10.0.1.0"},
		{"255.255.255.255", 0, "255.255.255.255"},
		{"::1", 1, "::2"},
		{"::ffff", 1, "::1:0"},
		{"fd00::1", -1, "fd00::"},
	}
	for _, tc := range tests {
		t.Run(tc.ip, func(t *testing.T) {
			got, err := IPAdd(tc.ip, tc.n)
			fatalOnErr(t, err)
			if got != tc.want {
				t.Errorf("IPAdd(%q, %d) = %q, want %q", tc.ip, tc.n, got, tc.want)
			}
		})
	}
	_, err := IPAdd("255.255.255.255", 1)
	wantErr(t, err)
	_, err = IPAdd("0.0.0.0", -1)
	wantErr(t, err)
}

// ---- EnumerateCIDR ----------------------------------------------------------

func TestEnumerateCIDR(t *testing.T) {
	got, err := EnumerateCIDR("10.0.0.0/24", 2)
	fatalOnErr(t, err)
	want := []string{"10.0.0.0/26", "10.0.0.64/26", "10.0.0.128/26", "10.0.0.192/26"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("[%d] got %q, want %q", i, got[i], want[i])
		}
	}

	_, err = EnumerateCIDR("10.0.0.0/24", 0)
	wantErr(t, err)
	_, err = EnumerateCIDR("10.0.0.0/24", 9) // 2^9=512 > 8 host bits
	wantErr(t, err)
	_, err = EnumerateCIDR("10.0.0.0/8", 17) // 2^17 > 65536
	wantErr(t, err)
}

// ---- CIDRWildcard -----------------------------------------------------------

func TestCIDRWildcard(t *testing.T) {
	tests := []struct{ cidr, want string }{
		{"10.0.0.0/24", "0.0.0.255"},
		{"10.0.0.0/16", "0.0.255.255"},
		{"10.0.0.0/8", "0.255.255.255"},
		{"10.0.0.1/32", "0.0.0.0"},
		{"0.0.0.0/0", "255.255.255.255"},
	}
	for _, tc := range tests {
		t.Run(tc.cidr, func(t *testing.T) {
			got, err := CIDRWildcard(tc.cidr)
			fatalOnErr(t, err)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
	_, err := CIDRWildcard("2001:db8::/32")
	wantErr(t, err)
}

// ---- CIDRUsableHostCount ----------------------------------------------------

func TestCIDRUsableHostCount(t *testing.T) {
	tests := []struct {
		cidr string
		want int64
	}{
		{"10.0.0.0/24", 254},
		{"10.0.0.0/16", 65534},
		{"10.0.0.0/30", 2},
		{"10.0.0.0/31", 2},                        // RFC 3021 point-to-point
		{"10.0.0.1/32", 1},                        // host route
		{"10.0.0.0/32", 1},                        // host route (network address form)
		{"2001:db8::/64", int64(^uint64(0) >> 1)}, // IPv6: all addresses usable (2^64, capped)
	}
	for _, tc := range tests {
		t.Run(tc.cidr, func(t *testing.T) {
			got, err := CIDRUsableHostCount(tc.cidr)
			fatalOnErr(t, err)
			if got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

// ---- CIDRsAreDisjoint -------------------------------------------------------

func TestCIDRsAreDisjoint(t *testing.T) {
	ok, err := CIDRsAreDisjoint([]string{"10.0.0.0/24", "10.0.1.0/24", "192.168.0.0/16"})
	fatalOnErr(t, err)
	if !ok {
		t.Error("expected disjoint")
	}

	ok, err = CIDRsAreDisjoint([]string{"10.0.0.0/8", "10.0.1.0/24"}) // contains
	fatalOnErr(t, err)
	if ok {
		t.Error("expected not disjoint (contains)")
	}

	ok, err = CIDRsAreDisjoint([]string{"10.0.0.0/24", "10.0.0.128/25"}) // overlap
	fatalOnErr(t, err)
	if ok {
		t.Error("expected not disjoint (overlap)")
	}

	ok, err = CIDRsAreDisjoint([]string{})
	fatalOnErr(t, err)
	if !ok {
		t.Error("empty list should be disjoint")
	}

	ok, err = CIDRsAreDisjoint([]string{"10.0.0.0/24"})
	fatalOnErr(t, err)
	if !ok {
		t.Error("single element should be disjoint")
	}
}

// ---- FindFreeCIDR -----------------------------------------------------------

func TestFindFreeCIDR(t *testing.T) {
	p, err := FindFreeCIDR([]string{"10.0.0.0/16"}, []string{}, 24)
	fatalOnErr(t, err)
	if p == nil || *p != "10.0.0.0/24" {
		t.Errorf("got %v, want 10.0.0.0/24", p)
	}

	p, err = FindFreeCIDR([]string{"10.0.0.0/16"}, []string{"10.0.0.0/24", "10.0.1.0/24"}, 24)
	fatalOnErr(t, err)
	if p == nil || *p != "10.0.2.0/24" {
		t.Errorf("got %v, want 10.0.2.0/24", p)
	}

	p, err = FindFreeCIDR([]string{"10.0.0.0/30"}, []string{"10.0.0.0/30"}, 24)
	fatalOnErr(t, err)
	if p != nil {
		t.Errorf("expected nil, got %q", *p)
	}

	_, err = FindFreeCIDR([]string{"10.0.0.0/8"}, []string{}, 200)
	wantErr(t, err)
}

// ---- NAT64 (RFC 6052) -------------------------------------------------------

func TestNAT64PrefixValid(t *testing.T) {
	valid := []string{
		"64:ff9b::/96",
		"64:ff9b:1::/48",
		"2001:db8::/32",
		"2001:db8::/40",
		"2001:db8::/48",
		"2001:db8::/56",
		"2001:db8::/64",
	}
	for _, p := range valid {
		t.Run("valid_"+p, func(t *testing.T) {
			got, err := NAT64PrefixValid(p)
			fatalOnErr(t, err)
			if !got {
				t.Errorf("%q should be valid", p)
			}
		})
	}

	invalid := []string{
		"10.0.0.0/8",            // IPv4
		"2001:db8::/33",         // wrong length
		"2001:db8::/128",        // wrong length
		"2001:db8:0:0:100::/64", // u-octet non-zero
	}
	for _, p := range invalid {
		t.Run("invalid_"+p, func(t *testing.T) {
			got, _ := NAT64PrefixValid(p)
			if got {
				t.Errorf("%q should be invalid", p)
			}
		})
	}
}

// Test vectors derived from RFC 6052 Section 2.4.
func TestNAT64Synthesize(t *testing.T) {
	tests := []struct {
		ipv4      string
		prefix    string
		wantHex   string
		wantMixed string
	}{
		{"192.0.2.1", "64:ff9b::/96", "64:ff9b::c000:201", "64:ff9b::192.0.2.1"},
		{"192.0.2.33", "2001:db8::/64", "2001:db8::c0:2:2100:0", "2001:db8::c0:2:33.0.0.0"},
	}
	for _, tc := range tests {
		t.Run(tc.ipv4+"_"+tc.prefix, func(t *testing.T) {
			gotHex, err := NAT64Synthesize(tc.ipv4, tc.prefix, false)
			fatalOnErr(t, err)
			if gotHex != tc.wantHex {
				t.Errorf("hex: got %q, want %q", gotHex, tc.wantHex)
			}
			gotMixed, err := NAT64Synthesize(tc.ipv4, tc.prefix, true)
			fatalOnErr(t, err)
			if gotMixed != tc.wantMixed {
				t.Errorf("mixed: got %q, want %q", gotMixed, tc.wantMixed)
			}
		})
	}
	_, err := NAT64Synthesize("not-an-ip", "64:ff9b::/96", false)
	wantErr(t, err)
	_, err = NAT64Synthesize("192.0.2.1", "64:ff9b::/33", false)
	wantErr(t, err)
	_, err = NAT64Synthesize("2001:db8::1", "64:ff9b::/96", false)
	wantErr(t, err)
}

func TestNAT64Extract(t *testing.T) {
	// No prefix arg: last-32-bits extraction, correct for /96.
	got, err := NAT64Extract("64:ff9b::c000:201", "")
	fatalOnErr(t, err)
	if got != "192.0.2.1" {
		t.Errorf("no-prefix extract: got %q", got)
	}

	// Round-trip for /96: synthesize then extract with no prefix arg.
	for _, ip := range []string{"192.0.2.1", "10.0.0.1", "203.0.113.255"} {
		syn, err := NAT64Synthesize(ip, "64:ff9b::/96", false)
		fatalOnErr(t, err)
		back, err := NAT64Extract(syn, "")
		fatalOnErr(t, err)
		if back != ip {
			t.Errorf("no-prefix round-trip %q: got %q via %q", ip, back, syn)
		}
	}

	// With prefix arg: uses RFC 6052 layout for non-/96.
	for _, prefix := range []string{"64:ff9b::/96", "2001:db8::/64"} {
		for _, ip := range []string{"192.0.2.1", "10.0.0.1"} {
			t.Run(prefix+"_"+ip, func(t *testing.T) {
				syn, err := NAT64Synthesize(ip, prefix, false)
				fatalOnErr(t, err)
				got, err := NAT64Extract(syn, prefix)
				fatalOnErr(t, err)
				if got != ip {
					t.Errorf("prefix round-trip: got %q, want %q (via %q)", got, ip, syn)
				}
			})
		}
	}
}

func TestNAT64SynthesizeCIDR(t *testing.T) {
	got, err := NAT64SynthesizeCIDR("192.0.2.0/24", "64:ff9b::/96", false)
	fatalOnErr(t, err)
	if got != "64:ff9b::c000:200/120" {
		t.Errorf("got %q", got)
	}

	gotMixed, err := NAT64SynthesizeCIDR("192.0.2.0/24", "64:ff9b::/96", true)
	fatalOnErr(t, err)
	if gotMixed != "64:ff9b::192.0.2.0/120" {
		t.Errorf("mixed: got %q", gotMixed)
	}

	_, err = NAT64SynthesizeCIDR("10.0.0.0/8", "2001:db8::/48", false)
	wantErr(t, err)
}

// ---- NPTv6 (RFC 6296) -------------------------------------------------------

func TestNPTv6Translate(t *testing.T) {
	internal := "fd00::/48"
	external := "2001:db8::/48"

	addresses := []string{
		"fd00::1",
		"fd00::dead:beef",
		"fd00::1:2:3:4",
	}
	extPrefix, _ := ParsePrefix(external)
	intPrefix, _ := ParsePrefix(internal)

	for _, addr := range addresses {
		t.Run(addr, func(t *testing.T) {
			ext, err := NPTv6Translate(addr, internal, external)
			fatalOnErr(t, err)

			extAddr, _ := ParseAddr(ext)
			if !extPrefix.Contains(extAddr) {
				t.Errorf("%q not in external prefix after translation", ext)
			}

			back, err := NPTv6Translate(ext, external, internal)
			fatalOnErr(t, err)
			if back != addr {
				t.Errorf("round-trip: got %q, want %q", back, addr)
			}

			// Verify the translation is in the internal prefix too
			backAddr, _ := ParseAddr(back)
			if !intPrefix.Contains(backAddr) {
				t.Errorf("%q not in internal prefix after reverse translation", back)
			}
		})
	}

	_, err := NPTv6Translate("2001:db8::1", "fd00::/48", "2001:db8::/48")
	wantErr(t, err)
	_, err = NPTv6Translate("fd00::1", "fd00::/32", "2001:db8::/32")
	wantErr(t, err)
}

// ---- Mixed notation ---------------------------------------------------------

func TestIPToMixedNotation(t *testing.T) {
	tests := []struct{ input, want string }{
		{"64:ff9b::c000:201", "64:ff9b::192.0.2.1"},
		{"::1", "::0.0.0.1"},
		{"::ffff:c0a8:101", "192.168.1.1"}, // IPv4-mapped unmaps to IPv4
		{"2001:db8::c000:201", "2001:db8::192.0.2.1"},
		{"192.0.2.1", "192.0.2.1"}, // IPv4 passthrough
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := IPToMixedNotation(tc.input)
			fatalOnErr(t, err)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestIPv4ToIPv4Mapped(t *testing.T) {
	tests := []struct{ ipv4, want string }{
		{"192.0.2.1", "::ffff:192.0.2.1"},
		{"0.0.0.0", "::ffff:0.0.0.0"},
		{"255.255.255.255", "::ffff:255.255.255.255"},
	}
	for _, tc := range tests {
		t.Run(tc.ipv4, func(t *testing.T) {
			got, err := IPv4ToIPv4Mapped(tc.ipv4)
			fatalOnErr(t, err)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
	_, err := IPv4ToIPv4Mapped("2001:db8::1")
	wantErr(t, err)
}

// ---- Parse error handling ---------------------------------------------------

func TestParseErrors(t *testing.T) {
	_, err := ParsePrefix("not-a-cidr")
	wantErr(t, err)
	_, err = ParseAddr("not-an-ip")
	wantErr(t, err)
	_, err = MergeCIDRs([]string{"bad"})
	wantErr(t, err)
	_, err = SubtractCIDRs([]string{"bad"}, []string{})
	wantErr(t, err)
	_, err = RangeToCIDRs("bad", "10.0.0.1")
	wantErr(t, err)
	_, err = CIDRWildcard("bad")
	wantErr(t, err)
	_, err = IPAdd("bad", 1)
	wantErr(t, err)
	_, err = NAT64Synthesize("bad", "64:ff9b::/96", false)
	wantErr(t, err)
	_, err = NAT64Extract("bad", "")
	wantErr(t, err)
	_, err = NPTv6Translate("bad", "fd00::/48", "2001:db8::/48")
	wantErr(t, err)
}

// ---- ExpandCIDR -------------------------------------------------------------

func TestExpandCIDR(t *testing.T) {
	got, err := ExpandCIDR("10.0.0.0/30")
	fatalOnErr(t, err)
	want := []string{"10.0.0.0", "10.0.0.1", "10.0.0.2", "10.0.0.3"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("[%d] got %q want %q", i, got[i], want[i])
		}
	}
	// /32 expands to one address
	got, err = ExpandCIDR("10.0.0.5/32")
	fatalOnErr(t, err)
	if len(got) != 1 || got[0] != "10.0.0.5" {
		t.Errorf("got %v", got)
	}
	// Too large
	_, err = ExpandCIDR("10.0.0.0/15")
	wantErr(t, err)
}

// ---- IPInCIDR / IPInCIDRs ---------------------------------------------------

func TestIPInCIDR(t *testing.T) {
	ok, err := IPInCIDR("10.0.0.5", "10.0.0.0/24")
	fatalOnErr(t, err)
	if !ok {
		t.Error("expected true")
	}
	ok, err = IPInCIDR("10.0.1.5", "10.0.0.0/24")
	fatalOnErr(t, err)
	if ok {
		t.Error("expected false")
	}
	// Network address is in the CIDR
	ok, err = IPInCIDR("10.0.0.0", "10.0.0.0/24")
	fatalOnErr(t, err)
	if !ok {
		t.Error("expected network address to be in CIDR")
	}
}

func TestCIDRsContainingIP(t *testing.T) {
	got, err := CIDRsContainingIP("10.0.1.5", []string{"10.0.0.0/24", "10.0.1.0/24", "192.168.0.0/16"})
	fatalOnErr(t, err)
	if len(got) != 1 || got[0] != "10.0.1.0/24" {
		t.Errorf("got %v", got)
	}
	// No match
	got, err = CIDRsContainingIP("1.2.3.4", []string{"10.0.0.0/8"})
	fatalOnErr(t, err)
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
	// Multiple matches (overlapping CIDRs)
	got, err = CIDRsContainingIP("10.0.1.5", []string{"10.0.0.0/8", "10.0.1.0/24"})
	fatalOnErr(t, err)
	if len(got) != 2 {
		t.Errorf("expected 2 matches, got %v", got)
	}
}

// ---- CIDRContains / CIDROverlaps --------------------------------------------

func TestCIDRContains(t *testing.T) {
	ok, err := CIDRContains("10.0.0.0/8", "10.1.2.0/24")
	fatalOnErr(t, err)
	if !ok {
		t.Error("expected cidr to contain subnet")
	}
	ok, err = CIDRContains("10.0.0.0/8", "10.1.2.3")
	fatalOnErr(t, err)
	if !ok {
		t.Error("expected cidr to contain ip")
	}
	ok, err = CIDRContains("10.0.0.0/24", "10.0.1.0/24")
	fatalOnErr(t, err)
	if ok {
		t.Error("expected false for non-contained subnet")
	}
	// Equal prefix: a CIDR contains itself
	ok, err = CIDRContains("10.0.0.0/24", "10.0.0.0/24")
	fatalOnErr(t, err)
	if !ok {
		t.Error("expected cidr to contain itself")
	}
}

func TestCIDROverlaps(t *testing.T) {
	ok, err := CIDROverlaps("10.0.0.0/24", "10.0.0.128/25")
	fatalOnErr(t, err)
	if !ok {
		t.Error("expected overlap")
	}
	ok, err = CIDROverlaps("10.0.0.0/24", "10.0.1.0/24")
	fatalOnErr(t, err)
	if ok {
		t.Error("expected no overlap")
	}
}

// ---- CIDRHostCount / CIDRFirstIP / CIDRLastIP / CIDRPrefixLength ------------

func TestCIDRInfoFunctions(t *testing.T) {
	// CIDRHostCount
	n, err := CIDRHostCount("10.0.0.0/24")
	fatalOnErr(t, err)
	if n != 256 {
		t.Errorf("CIDRHostCount /24 = %d, want 256", n)
	}
	n, err = CIDRHostCount("10.0.0.1/32")
	fatalOnErr(t, err)
	if n != 1 {
		t.Errorf("CIDRHostCount /32 = %d, want 1", n)
	}

	// CIDRFirstIP
	s, err := CIDRFirstIP("10.0.0.7/24")
	fatalOnErr(t, err)
	if s != "10.0.0.0" {
		t.Errorf("CIDRFirstIP = %q, want 10.0.0.0", s)
	}

	// CIDRLastIP
	s, err = CIDRLastIP("10.0.0.0/24")
	fatalOnErr(t, err)
	if s != "10.0.0.255" {
		t.Errorf("CIDRLastIP = %q, want 10.0.0.255", s)
	}

	// CIDRPrefixLength
	n, err = CIDRPrefixLength("10.0.0.0/23")
	fatalOnErr(t, err)
	if n != 23 {
		t.Errorf("CIDRPrefixLength = %d, want 23", n)
	}
}

// ---- IPVersion / CIDRVersion ------------------------------------------------

func TestVersionFunctions(t *testing.T) {
	n, err := IPVersion("192.168.1.1")
	fatalOnErr(t, err)
	if n != 4 {
		t.Errorf("IPVersion v4 = %d", n)
	}
	n, err = IPVersion("2001:db8::1")
	fatalOnErr(t, err)
	if n != 6 {
		t.Errorf("IPVersion v6 = %d", n)
	}
	n, err = CIDRVersion("10.0.0.0/8")
	fatalOnErr(t, err)
	if n != 4 {
		t.Errorf("CIDRVersion v4 = %d", n)
	}
	n, err = CIDRVersion("2001:db8::/32")
	fatalOnErr(t, err)
	if n != 6 {
		t.Errorf("CIDRVersion v6 = %d", n)
	}
}

// ---- IPIsPrivate / CIDRIsPrivate --------------------------------------------

func TestPrivateFunctions(t *testing.T) {
	privateIPs := []string{
		"10.0.0.1",    // RFC1918
		"172.16.0.1",  // RFC1918
		"192.168.0.1", // RFC1918
		"127.0.0.1",   // loopback
		"169.254.1.1", // link-local
		"100.64.0.1",  // RFC6598 CGNAT
		"::1",         // IPv6 loopback
		"fc00::1",     // IPv6 ULA
		"fe80::1",     // IPv6 link-local
	}
	for _, ip := range privateIPs {
		t.Run("private_"+ip, func(t *testing.T) {
			ok, err := IPIsPrivate(ip)
			fatalOnErr(t, err)
			if !ok {
				t.Errorf("%q should be private", ip)
			}
		})
	}
	publicIPs := []string{"8.8.8.8", "1.1.1.1", "203.0.113.1"}
	for _, ip := range publicIPs {
		t.Run("public_"+ip, func(t *testing.T) {
			ok, err := IPIsPrivate(ip)
			fatalOnErr(t, err)
			if ok {
				t.Errorf("%q should not be private", ip)
			}
		})
	}

	ok, err := CIDRIsPrivate("10.0.0.0/8")
	fatalOnErr(t, err)
	if !ok {
		t.Error("10/8 should be private")
	}
	ok, err = CIDRIsPrivate("8.0.0.0/8")
	fatalOnErr(t, err)
	if ok {
		t.Error("8/8 should not be private")
	}
	// A CIDR that spans a private/public boundary is not fully private
	ok, err = CIDRIsPrivate("0.0.0.0/0")
	fatalOnErr(t, err)
	if ok {
		t.Error("0/0 should not be private")
	}
}

// ---- FilterCIDRsByVersion ---------------------------------------------------

func TestFilterCIDRsByVersion(t *testing.T) {
	mixed := []string{"10.0.0.0/8", "172.16.0.0/12", "2001:db8::/32", "fd00::/8"}

	v4, err := FilterCIDRsByVersion(mixed, 4)
	fatalOnErr(t, err)
	if len(v4) != 2 || v4[0] != "10.0.0.0/8" || v4[1] != "172.16.0.0/12" {
		t.Errorf("v4 filter: got %v", v4)
	}
	v6, err := FilterCIDRsByVersion(mixed, 6)
	fatalOnErr(t, err)
	if len(v6) != 2 || v6[0] != "2001:db8::/32" || v6[1] != "fd00::/8" {
		t.Errorf("v6 filter: got %v", v6)
	}
	_, err = FilterCIDRsByVersion(mixed, 5)
	wantErr(t, err)
}

// ---- NAT64SynthesizeCIDRs ---------------------------------------------------

func TestNAT64SynthesizeCIDRs(t *testing.T) {
	got, err := NAT64SynthesizeCIDRs(
		[]string{"192.0.2.0/24", "198.51.100.0/24"},
		"64:ff9b::/96",
		true,
	)
	fatalOnErr(t, err)
	want := []string{"64:ff9b::192.0.2.0/120", "64:ff9b::198.51.100.0/120"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("[%d] got %q want %q", i, got[i], want[i])
		}
	}
}

// ---- NAT64 all prefix lengths (/32, /40, /48, /56) round-trip --------------

func TestNAT64AllPrefixLengths(t *testing.T) {
	// Use a distinct IPv4 address that exercises all 4 bytes distinctly.
	ipv4 := "192.168.10.99" // 0xC0, 0xA8, 0x0A, 0x63

	prefixes := []string{
		"2001:db8::/32",
		"2001:db8::/40",
		"2001:db8::/48",
		"2001:db8::/56",
		"2001:db8::/64",
		"64:ff9b::/96",
	}
	for _, prefix := range prefixes {
		t.Run(prefix, func(t *testing.T) {
			syn, err := NAT64Synthesize(ipv4, prefix, false)
			fatalOnErr(t, err)
			// /96 can use the no-arg form; others need the prefix for layout.
			extractPrefix := prefix
			if prefix == "64:ff9b::/96" {
				extractPrefix = ""
			}
			back, err := NAT64Extract(syn, extractPrefix)
			fatalOnErr(t, err)
			if back != ipv4 {
				t.Errorf("round-trip via %s: got %q, want %q (synthesized: %q)",
					prefix, back, ipv4, syn)
			}
		})
	}
}

// ---- IPSubtract -------------------------------------------------------------

func TestIPSubtract(t *testing.T) {
	tests := []struct {
		a, b string
		want int64
	}{
		{"10.0.0.10", "10.0.0.1", 9},
		{"10.0.0.1", "10.0.0.10", -9},
		{"10.0.0.5", "10.0.0.5", 0},
		{"10.0.1.0", "10.0.0.255", 1}, // crosses byte boundary
		{"255.255.255.255", "0.0.0.0", 4294967295},
		{"::10", "::1", 15},
		{"::1", "::10", -15},
	}
	for _, tc := range tests {
		t.Run(tc.a+"-"+tc.b, func(t *testing.T) {
			got, err := IPSubtract(tc.a, tc.b)
			fatalOnErr(t, err)
			if got != tc.want {
				t.Errorf("IPSubtract(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
	// Mixed family error
	_, err := IPSubtract("10.0.0.1", "::1")
	wantErr(t, err)
	// IPv6 overflow (high 64 bits differ)
	_, err = IPSubtract("2001:db8::1", "fd00::1")
	wantErr(t, err)
}

// ---- Specific custom-logic edge cases not yet covered -----------------------

func TestCIDRHostCountLargeIPv6(t *testing.T) {
	// hostBits >= 63 cap (e.g. /0 IPv6 = 2^128 hosts)
	n, err := CIDRHostCount("::/0")
	fatalOnErr(t, err)
	if n != int64(^uint64(0)>>1) {
		t.Errorf("expected MaxInt64 for ::/0, got %d", n)
	}
	n, err = CIDRHostCount("::/1")
	fatalOnErr(t, err)
	if n != int64(^uint64(0)>>1) {
		t.Errorf("expected MaxInt64 for ::/1, got %d", n)
	}
}

func TestIPToMixedNotationNoCompression(t *testing.T) {
	// All 6 hex groups non-zero → no :: compression.
	// 2001:db8:cafe:babe:dead:beef with v4 1.2.3.4
	got, err := IPToMixedNotation("2001:db8:cafe:babe:dead:beef:102:304")
	fatalOnErr(t, err)
	if got != "2001:db8:cafe:babe:dead:beef:1.2.3.4" {
		t.Errorf("got %q", got)
	}
	// Single zero group (bestLen=1, no compression)
	got, err = IPToMixedNotation("2001:db8:0:1:2:3:102:304")
	fatalOnErr(t, err)
	if got != "2001:db8:0:1:2:3:1.2.3.4" {
		t.Errorf("got %q", got)
	}
}

func TestIPAddIPv6Carry(t *testing.T) {
	// Adding 1 to MaxUint64 in the low half should carry into the high half.
	// Address with lo = MaxUint64: the address is 0:0:0:0:ffff:ffff:ffff:ffff
	// = "::ffff:ffff:ffff:ffff"
	got, err := IPAdd("::ffff:ffff:ffff:ffff", 1)
	fatalOnErr(t, err)
	// hi was 0, lo was MaxUint64. Adding 1: newLo=0, carry=1, newHi=1.
	// Address: hi=1, lo=0 = 0:0:0:1:: = "0:0:0:1::" compressed
	if got != "0:0:0:1::" {
		t.Errorf("carry: got %q, want 0:0:0:1::", got)
	}
	// Overflow: adding 1 to the max IPv6 address should error
	_, err = IPAdd("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff", 1)
	wantErr(t, err)
}

func TestIPSubtractIPv6Borrow(t *testing.T) {
	// MaxInt64 = 0x7fffffffffffffff.
	// "::7fff:ffff:ffff:ffff" has lo = MaxInt64, hi = 0.
	// Subtracting "::" (lo=0) gives diff = MaxInt64 exactly — should succeed.
	got, err := IPSubtract("::7fff:ffff:ffff:ffff", "::")
	fatalOnErr(t, err)
	if got != int64(^uint64(0)>>1) {
		t.Errorf("got %d, want MaxInt64", got)
	}
	// "::8000:0:0:0" has lo = MaxInt64+1 = 0x8000000000000000.
	// diff = MaxInt64+1 > MaxInt64 — should error.
	_, err = IPSubtract("::8000:0:0:0", "::")
	wantErr(t, err)
	// High bits differ — always errors.
	_, err = IPSubtract("fd00::1", "2001:db8::1")
	wantErr(t, err)
}
