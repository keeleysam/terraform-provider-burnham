package cel

import (
	"math"
	"reflect"
	"testing"
)

func TestEval(t *testing.T) {
	cases := []struct {
		name string
		expr string
		vars map[string]any
		want any
	}{
		{"arithmetic", "1 + 2 * 3", nil, int64(7)},
		{"string concat", `"a" + "b" + "c"`, nil, "abc"},
		{"comparison with var", "x > 5", map[string]any{"x": int64(10)}, true},
		{"membership", `"admin" in roles`, map[string]any{"roles": []any{"user", "admin"}}, true},
		{"map macro", "items.map(i, i * 2)", map[string]any{"items": []any{int64(1), int64(2), int64(3)}}, []any{int64(2), int64(4), int64(6)}},
		{"filter size", "items.filter(i, i > 1).size()", map[string]any{"items": []any{int64(1), int64(2), int64(3)}}, int64(2)},
		{"nested var field", `req.tier == "prod"`, map[string]any{"req": map[string]any{"tier": "prod"}}, true},
		{"map literal", `{"k": 1, "j": 2}`, nil, map[string]any{"k": int64(1), "j": int64(2)}},
		{"ternary", `x > 5 ? "big" : "small"`, map[string]any{"x": int64(2)}, "small"},
		{"has macro", `has(m.a) ? m.a : "default"`, map[string]any{"m": map[string]any{"a": "present"}}, "present"},
		{"timestamp rfc3339", `timestamp("2026-01-01T00:00:00Z")`, nil, "2026-01-01T00:00:00Z"},
		{"duration seconds string", `duration("1h30m")`, nil, "5400s"},
		{"bytes base64", `b"hi"`, nil, "aGk="},
		{"string ext lib", `"Hello".lowerAscii()`, nil, "hello"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Eval(tc.expr, tc.vars, defaultEvalOptions())
			if err != nil {
				t.Fatalf("Eval(%q) error: %v", tc.expr, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Eval(%q) = %#v, want %#v", tc.expr, got, tc.want)
			}
		})
	}
}

func TestEvalErrors(t *testing.T) {
	cases := []struct {
		name string
		expr string
		vars map[string]any
	}{
		{"undeclared variable", "nope + 1", nil},
		{"parse error", "1 +", nil},
		{"unknown function", `inIpRange(ip, ["10.0.0.0/8"])`, map[string]any{"ip": "10.1.2.3"}},
		{"eval error division by zero", "1 / 0", nil},
		{"non-string map key", `{1: "a"}`, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Eval(tc.expr, tc.vars, defaultEvalOptions()); err == nil {
				t.Fatalf("Eval(%q) = nil error, want error", tc.expr)
			}
		})
	}
}

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

func TestEvalResultFormatOverrides(t *testing.T) {
	ts, err := Eval(`timestamp("2026-01-01T00:00:00Z")`, nil, evalOptions{tsFormat: "unix", durFormat: "string", bytesFormat: "base64"})
	if err != nil || ts != int64(1767225600) {
		t.Fatalf("unix timestamp = %#v, err %v", ts, err)
	}
	d, err := Eval(`duration("1h30m")`, nil, evalOptions{tsFormat: "rfc3339", durFormat: "go", bytesFormat: "base64"})
	if err != nil || d != "1h30m0s" {
		t.Fatalf("go duration = %#v, err %v", d, err)
	}
	b, err := Eval(`b"hi"`, nil, evalOptions{tsFormat: "rfc3339", durFormat: "string", bytesFormat: "hex"})
	if err != nil || b != "6869" {
		t.Fatalf("hex bytes = %#v, err %v", b, err)
	}
}
