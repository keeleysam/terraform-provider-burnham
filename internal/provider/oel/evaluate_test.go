package oel

import (
	"math"
	"testing"
)

// TestNodeToAttrNonFinite guards against NaN and +/-Inf reaching types.NumberValue, which cannot represent them:
// NaN panics in big.NewFloat and infinities silently emit an invalid plan value. Both must be a clean error instead.
func TestNodeToAttrNonFinite(t *testing.T) {
	cases := []struct {
		name string
		in   float64
	}{
		{"NaN", math.NaN()},
		{"positive infinity", math.Inf(1)},
		{"negative infinity", math.Inf(-1)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := nodeToAttr(tc.in); err == nil {
				t.Fatalf("nodeToAttr(%v) = nil error, want error", tc.in)
			}
		})
	}
}

func TestEvaluate(t *testing.T) {
	cases := []struct {
		name string
		expr string
		ctx  EvalContext
		want any
	}{
		{
			"path equality true",
			`user.department == "Sales"`,
			EvalContext{UserProfile: map[string]any{"department": "Sales"}},
			true,
		},
		{
			"path equality false",
			`user.department == "Sales"`,
			EvalContext{UserProfile: map[string]any{"department": "Engineering"}},
			false,
		},
		{
			"string function on profile",
			`String.stringContains(user.department, "Sale")`,
			EvalContext{UserProfile: map[string]any{"department": "Sales"}},
			true,
		},
		{
			"numeric comparison",
			`user.salary > 1000000`,
			EvalContext{UserProfile: map[string]any{"salary": 2000000}},
			true,
		},
		{
			"ternary",
			`user.department == "Sales" ? "yes" : "no"`,
			EvalContext{UserProfile: map[string]any{"department": "Sales"}},
			"yes",
		},
		{
			"is member of any group true",
			`isMemberOfAnyGroup("00g1")`,
			EvalContext{GroupIDs: []string{"00g1"}},
			true,
		},
		{
			"is member of any group false",
			`isMemberOfAnyGroup("00g1")`,
			EvalContext{GroupIDs: []string{"00g2"}},
			false,
		},
		{
			"is member of group name via group data",
			`isMemberOfGroupName("Engineering")`,
			EvalContext{
				GroupIDs:  []string{"00g1"},
				GroupData: map[string]any{"00g1": map[string]any{"profile": map[string]any{"name": "Engineering"}}},
			},
			true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Evaluate(tc.expr, tc.ctx)
			if err != nil {
				t.Fatalf("Evaluate(%q) error: %v", tc.expr, err)
			}
			if got != tc.want {
				t.Fatalf("Evaluate(%q) = %#v, want %#v", tc.expr, got, tc.want)
			}
		})
	}
}

func TestEvaluateErrors(t *testing.T) {
	cases := []struct {
		name string
		expr string
	}{
		{"syntax error", `user.department ==`},
		{"unsupported method call", `user.getInternalProperty("status")`},
		{"unsupported isMemberOf object form", `user.isMemberOf({'group.profile.name': 'x'})`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Evaluate(tc.expr, EvalContext{}); err == nil {
				t.Fatalf("Evaluate(%q) = nil error, want error", tc.expr)
			}
		})
	}
}
