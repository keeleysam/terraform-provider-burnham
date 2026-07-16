package cedar

import (
	"fmt"
	"strings"
	"testing"
)

// TestDecodeEncodeLargeInteger guards against widening Cedar Long literals to
// float64 during decode, which would silently corrupt integers beyond 2^53.
func TestDecodeEncodeLargeInteger(t *testing.T) {
	src := `permit (principal, action, resource) when { principal.n == 9007199254740993 };`
	want := canonicalSingle(t, src)
	tree, err := Decode(src)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	got, err := Encode(tree)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if got != want {
		t.Fatalf("large integer lost precision:\n want %q\n got  %q", want, got)
	}
	if !strings.Contains(got, "9007199254740993") {
		t.Errorf("expected the exact integer, got %q", got)
	}
}

// TestEncodeRecordDeterministic guards against non-deterministic record-literal
// key ordering. cedar-go builds a record node's element slice by iterating a Go
// map (recordJSON.ToNode), so the EST JSON -> DSL step used to emit keys in
// random order. A plan-time function that returns different bytes for identical
// input causes perpetual Terraform diffs.
func TestEncodeRecordDeterministic(t *testing.T) {
	src := `permit (principal, action, resource) when { context.x == {a:1, b:2, c:3, d:4, e:5} };`
	tree, err := Decode(src)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	first, err := Encode(tree)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	for i := 0; i < 200; i++ {
		got, err := Encode(tree)
		if err != nil {
			t.Fatalf("Encode: %v", err)
		}
		if got != first {
			t.Fatalf("Encode not deterministic across calls:\n first: %q\n got:   %q", first, got)
		}
	}
	// The source keys are already in sorted order, so the canonical (sorted)
	// encoding must be byte-identical to what Format produces for the policy.
	want, err := Format(src)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	if first != want {
		t.Fatalf("Encode does not match Format:\n want: %q\n got:  %q", want, first)
	}
}

// TestDecodeMultiPolicyErrors: cedardecode handles a single policy; a document
// with several must error rather than silently keep the first.
func TestDecodeMultiPolicyErrors(t *testing.T) {
	_, err := Decode(`permit (principal, action, resource);
forbid (principal, action, resource) when { resource.private };`)
	if err == nil {
		t.Fatal("Decode of a multi-policy document should error")
	}
}

// TestFormatManyPoliciesOrderAndIdempotent: 10+ policies must stay in input
// order and format idempotently (marshaling the whole set sorts IDs
// lexicographically, putting policy10 before policy2).
func TestFormatManyPoliciesOrderAndIdempotent(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 12; i++ {
		fmt.Fprintf(&b, "permit (principal, action, resource) when { resource.n == %d };\n", i)
	}
	out, err := Format(b.String())
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	out2, err := Format(out)
	if err != nil {
		t.Fatalf("Format (second pass): %v", err)
	}
	if out != out2 {
		t.Fatalf("Format is not idempotent for 12 policies")
	}
	i9, i10, i11 := strings.Index(out, "== 9 "), strings.Index(out, "== 10 "), strings.Index(out, "== 11 ")
	if !(i9 >= 0 && i9 < i10 && i10 < i11) {
		t.Fatalf("policy order not preserved (9@%d, 10@%d, 11@%d)", i9, i10, i11)
	}
}
