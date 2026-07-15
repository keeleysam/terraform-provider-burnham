package promql

import (
	"errors"
	"testing"
)

// rateMatrix is a reusable sub-tree: rate(http_requests_total[5m]).
func rateMatrix() map[string]any {
	return map[string]any{"call": map[string]any{
		"func": "rate",
		"args": []any{map[string]any{"matrixSelector": map[string]any{"name": "http_requests_total", "range": "5m"}}},
	}}
}

// TestEncode builds a tree for each case and asserts it produces the same
// canonical query that Format yields for the equivalent hand-written query, so
// expectations are not hand-transcribed.
func TestEncode(t *testing.T) {
	cases := []struct {
		name string
		tree any
		want string // an equivalent PromQL query; the expected output is Format(want)
	}{
		{"bare metric", map[string]any{"vectorSelector": map[string]any{"name": "up"}}, `up`},
		{
			"selector with matchers",
			map[string]any{"vectorSelector": map[string]any{"name": "http_requests_total", "matchers": []any{
				map[string]any{"name": "job", "type": "=", "value": "apiserver"},
				map[string]any{"name": "code", "type": "=~", "value": "5.."},
			}}},
			`http_requests_total{job="apiserver", code=~"5.."}`,
		},
		{"offset", map[string]any{"vectorSelector": map[string]any{"name": "http_requests_total", "offset": "5m"}}, `http_requests_total offset 5m`},
		{"rate over matrix", rateMatrix(), `rate(http_requests_total[5m])`},
		{
			"sum by",
			map[string]any{"aggregation": map[string]any{"op": "sum", "by": []any{"job"}, "expr": rateMatrix()}},
			`sum by (job) (rate(http_requests_total[5m]))`,
		},
		{
			"topk with param",
			map[string]any{"aggregation": map[string]any{"op": "topk", "param": 5, "expr": rateMatrix()}},
			`topk(5, rate(http_requests_total[5m]))`,
		},
		{
			"binary comparison",
			map[string]any{"binaryExpr": map[string]any{"op": ">",
				"lhs": map[string]any{"aggregation": map[string]any{"op": "sum", "by": []any{"job"}, "expr": rateMatrix()}},
				"rhs": 0.05,
			}},
			`sum by (job) (rate(http_requests_total[5m])) > 0.05`,
		},
		{
			"binary with matching and group_left",
			map[string]any{"binaryExpr": map[string]any{"op": "/",
				"lhs":        map[string]any{"vectorSelector": map[string]any{"name": "errors"}},
				"rhs":        map[string]any{"vectorSelector": map[string]any{"name": "requests"}},
				"on":         []any{"job"},
				"group_left": []any{"instance"},
			}},
			`errors / on (job) group_left (instance) requests`,
		},
		{
			"subquery inside a call",
			map[string]any{"call": map[string]any{"func": "max_over_time", "args": []any{
				map[string]any{"subquery": map[string]any{"expr": rateMatrix(), "range": "30m", "step": "1m"}},
			}}},
			`max_over_time(rate(http_requests_total[5m])[30m:1m])`,
		},
		{
			"label_replace with string args",
			map[string]any{"call": map[string]any{"func": "label_replace", "args": []any{
				map[string]any{"vectorSelector": map[string]any{"name": "up", "matchers": []any{map[string]any{"name": "job", "type": "=", "value": "api"}}}},
				"service", "$1", "job", "(.*)",
			}}},
			`label_replace(up{job="api"}, "service", "$1", "job", "(.*)")`,
		},
		{"raw escape", map[string]any{"raw": `histogram_quantile(0.95, sum by (le) (rate(x[5m])))`}, `histogram_quantile(0.95, sum by (le) (rate(x[5m])))`},
		{"paren", map[string]any{"paren": map[string]any{"binaryExpr": map[string]any{"op": "+", "lhs": 1, "rhs": 2}}}, `(1 + 2)`},
		{"neg", map[string]any{"neg": map[string]any{"vectorSelector": map[string]any{"name": "up"}}}, `-up`},
		{"pos", map[string]any{"pos": map[string]any{"vectorSelector": map[string]any{"name": "up"}}}, `+up`},
		{
			"subquery with at",
			map[string]any{"subquery": map[string]any{"range": "10m", "step": "1m", "at": 100,
				"expr": rateMatrix(),
			}},
			`rate(http_requests_total[5m])[10m:1m] @ 100`,
		},
		{
			"atan2 operator",
			map[string]any{"binaryExpr": map[string]any{"op": "atan2",
				"lhs": map[string]any{"vectorSelector": map[string]any{"name": "up"}},
				"rhs": map[string]any{"vectorSelector": map[string]any{"name": "down"}},
			}},
			`up atan2 down`,
		},
		{"negative offset", map[string]any{"vectorSelector": map[string]any{"name": "http_requests_total", "offset": "-5m"}}, `http_requests_total offset -5m`},
		{"at unix timestamp", map[string]any{"vectorSelector": map[string]any{"name": "up", "at": 1609746000}}, `up @ 1609746000`},
		{"at start", map[string]any{"vectorSelector": map[string]any{"name": "up", "at": "start"}}, `up @ start()`},
		{"at end", map[string]any{"vectorSelector": map[string]any{"name": "up", "at": "end"}}, `up @ end()`},
		{
			"bool modifier",
			map[string]any{"binaryExpr": map[string]any{"op": ">", "bool": true,
				"lhs": map[string]any{"vectorSelector": map[string]any{"name": "up"}},
				"rhs": 1,
			}},
			`up > bool 1`,
		},
		{
			"count_values",
			map[string]any{"aggregation": map[string]any{"op": "count_values", "param": "version",
				"expr": map[string]any{"vectorSelector": map[string]any{"name": "build_info"}},
			}},
			`count_values("version", build_info)`,
		},
		{
			"group by",
			map[string]any{"aggregation": map[string]any{"op": "group", "by": []any{"job"},
				"expr": map[string]any{"vectorSelector": map[string]any{"name": "up"}},
			}},
			`group by (job) (up)`,
		},
		{
			"binary with ignoring",
			map[string]any{"binaryExpr": map[string]any{"op": "/",
				"lhs":      map[string]any{"vectorSelector": map[string]any{"name": "errors"}},
				"rhs":      map[string]any{"vectorSelector": map[string]any{"name": "requests"}},
				"ignoring": []any{"instance"},
			}},
			`errors / ignoring (instance) requests`,
		},
		{
			"unless set operator",
			map[string]any{"binaryExpr": map[string]any{"op": "unless",
				"lhs": map[string]any{"vectorSelector": map[string]any{"name": "up"}},
				"rhs": map[string]any{"vectorSelector": map[string]any{"name": "down"}},
			}},
			`up unless down`,
		},
		{
			"binary with group_right",
			map[string]any{"binaryExpr": map[string]any{"op": "/",
				"lhs":         map[string]any{"vectorSelector": map[string]any{"name": "errors"}},
				"rhs":         map[string]any{"vectorSelector": map[string]any{"name": "requests"}},
				"on":          []any{"job"},
				"group_right": []any{"instance"},
			}},
			`errors / on (job) group_right (instance) requests`,
		},
		{
			"stepless subquery",
			map[string]any{"subquery": map[string]any{"range": "5m",
				"expr": map[string]any{"vectorSelector": map[string]any{"name": "up"}},
			}},
			`up[5m:]`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want, err := Format(tc.want, false)
			if err != nil {
				t.Fatalf("Format(%q) oracle error: %v", tc.want, err)
			}
			got, err := Encode(tc.tree)
			if err != nil {
				t.Fatalf("Encode error: %v", err)
			}
			if got != want {
				t.Fatalf("Encode = %q, want %q", got, want)
			}
			if !IsValid(got) {
				t.Errorf("encoded output is not valid PromQL: %q", got)
			}
		})
	}
}

// TestEncodeInvalidOutput covers the second validation stage: a tree that builds
// into an AST fine but whose serialized form fails the parser's type check, so
// Encode returns errInvalidOutput rather than emitting an invalid query. Here
// rate() is applied to an instant vector, which the parser rejects on re-parse.
func TestEncodeInvalidOutput(t *testing.T) {
	tree := map[string]any{"call": map[string]any{"func": "rate", "args": []any{
		map[string]any{"vectorSelector": map[string]any{"name": "up"}},
	}}}
	_, err := Encode(tree)
	if err == nil {
		t.Fatal("Encode of rate() over an instant vector should error")
	}
	if !errors.Is(err, errInvalidOutput) {
		t.Fatalf("expected errInvalidOutput, got %v", err)
	}
}

func TestEncodeErrors(t *testing.T) {
	cases := []struct {
		name string
		tree any
	}{
		{"unknown function", map[string]any{"call": map[string]any{"func": "notafunc", "args": []any{}}}},
		{"unknown node key", map[string]any{"bogus": 1}},
		{"unknown aggregation op", map[string]any{"aggregation": map[string]any{"op": "median", "expr": rateMatrix()}}},
		{"matrixSelector without range", map[string]any{"matrixSelector": map[string]any{"name": "x"}}},
		{"bad matcher type", map[string]any{"vectorSelector": map[string]any{"name": "x", "matchers": []any{map[string]any{"name": "a", "type": "?", "value": "b"}}}}},
		{"multi-key object", map[string]any{"vectorSelector": map[string]any{"name": "x"}, "raw": "up"}},
		{"experimental function", map[string]any{"call": map[string]any{"func": "mad_over_time", "args": []any{rateMatrix()}}}},
		{"at out of range", map[string]any{"vectorSelector": map[string]any{"name": "up", "at": 1e300}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Encode(tc.tree); err == nil {
				t.Fatalf("Encode(%v) = nil error, want error", tc.tree)
			}
		})
	}
}
