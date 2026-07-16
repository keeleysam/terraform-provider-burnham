package cedar

import (
	"strings"
	"testing"
)

// TestDeepNestingDoesNotCrash guards the cedarvalidate/cedarformat/cedardecode/cedarevaluate contract that pathological input yields a normal error (validate returns false) instead of overflowing the goroutine stack and aborting the process. A policy whose condition nests hundreds of thousands of parentheses used to crash the provider with "fatal error: stack overflow" because the recursive-descent parser recurses once per nesting level.
func TestDeepNestingDoesNotCrash(t *testing.T) {
	deep := "permit (principal, action, resource) when {" + strings.Repeat("(", 700000) + "1" + strings.Repeat(")", 700000) + "};"
	if IsValid(deep) {
		t.Error("IsValid: deeply nested input should be reported invalid, not crash")
	}
	if _, err := Format(deep); err == nil {
		t.Error("Format: deeply nested input should return an error, not crash")
	}
	if _, err := Decode(deep); err == nil {
		t.Error("Decode: deeply nested input should return an error, not crash")
	}
	if _, err := Evaluate(deep, Request{}); err == nil {
		t.Error("Evaluate: deeply nested input should return an error, not crash")
	}
}
