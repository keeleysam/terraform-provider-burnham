package oel

import (
	"strings"
	"testing"
)

// TestDeepNestingDoesNotCrash guards the oelvalidate/oelformat/oeldecode/oelevaluate contract that pathological input yields a normal error (validate returns false) instead of overflowing the goroutine stack and aborting the process. A ~1.4MB string of nested parentheses used to crash the provider with "fatal error: stack overflow" because the recursive-descent parser recurses once per nesting level.
func TestDeepNestingDoesNotCrash(t *testing.T) {
	deep := strings.Repeat("(", 700000) + "1" + strings.Repeat(")", 700000)
	if IsValid(deep) {
		t.Error("IsValid: deeply nested input should be reported invalid, not crash")
	}
	if _, err := Format(deep); err == nil {
		t.Error("Format: deeply nested input should return an error, not crash")
	}
	if _, err := Decode(deep); err == nil {
		t.Error("Decode: deeply nested input should return an error, not crash")
	}
	if _, err := Evaluate(deep, EvalContext{}); err == nil {
		t.Error("Evaluate: deeply nested input should return an error, not crash")
	}
}
