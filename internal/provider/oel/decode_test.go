package oel

import "testing"

// TestRoundTripCorpus is the strongest coverage check: for every documented
// expression in the corpus, decoding then re-encoding reproduces the canonical
// form. This proves oelencode and oeldecode are inverses across the entire
// documented Okta EL surface.
func TestRoundTripCorpus(t *testing.T) {
	for _, e := range readCorpus(t) {
		want, err := Format(e.expr)
		if err != nil {
			t.Errorf("[%s] Format(%q): %v", e.section, e.expr, err)
			continue
		}
		tree, err := Decode(e.expr)
		if err != nil {
			t.Errorf("[%s] Decode(%q): %v", e.section, e.expr, err)
			continue
		}
		got, err := Encode(tree)
		if err != nil {
			t.Errorf("[%s] Encode(Decode(%q)): %v", e.section, e.expr, err)
			continue
		}
		if got != want {
			t.Errorf("[%s] round-trip mismatch\n  in:   %q\n  want: %q\n  got:  %q", e.section, e.expr, want, got)
		}
	}
}

// TestDecodeShapes checks a few decoded trees have the expected surface shape.
func TestDecodeShapes(t *testing.T) {
	got, err := Decode(`user.department == "Sales"`)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	cmp, ok := got.(map[string]any)["=="]
	if !ok {
		t.Fatalf("expected a == node, got %#v", got)
	}
	if _, ok := cmp.([]any); !ok {
		t.Fatalf("== operands should be a list, got %#v", cmp)
	}

	got, err = Decode(`user.getInternalProperty("status")`)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	call, ok := got.(map[string]any)["call"].(map[string]any)
	if !ok {
		t.Fatalf("expected a call node, got %#v", got)
	}
	if call["method"] != "getInternalProperty" {
		t.Fatalf("expected method getInternalProperty, got %#v", call)
	}
}
