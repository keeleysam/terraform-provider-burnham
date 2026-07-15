package cedar

import (
	"testing"

	cedargo "github.com/cedar-policy/cedar-go"
)

// canonicalSingle returns the canonical single-policy DSL form of s, straight from cedar-go, so round-trip expectations are not hand-transcribed.
func canonicalSingle(t *testing.T, s string) string {
	t.Helper()
	var p cedargo.Policy
	if err := p.UnmarshalCedar([]byte(s)); err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return string(p.MarshalCedar())
}

// TestEncodeDecodeRoundTrip decodes each official example policy to its EST and
// re-encodes it, asserting it reproduces the canonical DSL and re-parses.
func TestEncodeDecodeRoundTrip(t *testing.T) {
	policies := []string{
		`permit (principal == User::"alice", action == Action::"view", resource == Photo::"VacationPhoto94.jpg");`,
		`permit (principal, action == Action::"editPhoto", resource) when { resource.owner == principal };`,
		`forbid (principal, action, resource) when { resource.private } unless { principal == resource.owner };`,
		`@id("free-content-access") permit (principal is FreeMember, action == Action::"watch", resource) when { resource.isFree };`,
		`permit (principal, action == Action::"GetList", resource) when { principal in resource.readers || principal in resource.editors };`,
		`permit (principal, action in [Action::"add_reader", Action::"add_admin"], resource) when { principal in resource.admins };`,
	}
	for _, src := range policies {
		want := canonicalSingle(t, src)
		tree, err := Decode(src)
		if err != nil {
			t.Errorf("Decode(%q): %v", src, err)
			continue
		}
		got, err := Encode(tree)
		if err != nil {
			t.Errorf("Encode(Decode(%q)): %v", src, err)
			continue
		}
		if got != want {
			t.Errorf("round-trip mismatch\n  in:   %q\n  want: %q\n  got:  %q", src, want, got)
		}
		if !IsValid(got) {
			t.Errorf("round-tripped output is not valid Cedar: %q", got)
		}
	}
}
