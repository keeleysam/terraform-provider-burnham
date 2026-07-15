package promql

import (
	"reflect"
	"testing"
)

// decodeRoundTripQueries exercise every node type Decode emits. For each, the
// contract is Encode(Decode(q)) == Format(q): decoding to the data tree and
// re-encoding yields the canonical query, so promqldecode and promqlencode are
// inverses on canonical forms.
var decodeRoundTripQueries = []string{
	`up`,
	`http_requests_total{code=~"5..", job="api"}`,
	`{__name__=~"up.*", job="api"}`,
	`rate(http_requests_total[5m])`,
	`sum by (job) (rate(http_requests_total[5m]))`,
	`sum without (instance) (node_memory_MemFree_bytes)`,
	`topk(5, rate(http_requests_total[5m]))`,
	`count_values("version", build_info)`,
	`group by (job) (up)`,
	`quantile(0.9, up)`,
	`sum by (job) (rate(http_requests_total[5m])) > 0.05`,
	`errors / on (job) group_left (instance) requests`,
	`errors / ignoring (instance) group_right (env) requests`,
	`errors / ignoring (instance) requests`,
	`up and on (job) down`,
	`up unless down`,
	`up > bool 1`,
	`up atan2 down`,
	`max_over_time(rate(http_requests_total[5m])[30m:1m])`,
	`up[5m:]`,
	`rate(http_requests_total[5m])[10m:1m] @ 100`,
	`label_replace(up{job="api"}, "service", "$1", "job", "(.*)")`,
	`http_requests_total offset 5m`,
	`http_requests_total offset -5m`,
	`up @ 1609746000`,
	`up @ start()`,
	`up @ end()`,
	`(1 + 2) * 3`,
	`-up`,
	`+up`,
	`x - -0`,
	`histogram_quantile(0.95, sum by (le) (rate(http_request_duration_seconds_bucket[5m])))`,
	`1.5`,
	`"a string"`,
	`time()`,
}

func TestDecodeRoundTrip(t *testing.T) {
	for _, q := range decodeRoundTripQueries {
		t.Run(q, func(t *testing.T) {
			want, err := Format(q, false)
			if err != nil {
				t.Fatalf("Format(%q): %v", q, err)
			}
			tree, err := Decode(q)
			if err != nil {
				t.Fatalf("Decode(%q): %v", q, err)
			}
			got, err := Encode(tree)
			if err != nil {
				t.Fatalf("Encode(Decode(%q)): %v", q, err)
			}
			if got != want {
				t.Fatalf("Encode(Decode(%q)) = %q, want %q\n tree: %#v", q, got, want, tree)
			}
		})
	}
}

// TestDecodeStructure locks the emitted notation for a few representative
// queries, not just the round-trip property.
func TestDecodeStructure(t *testing.T) {
	cases := []struct {
		query string
		want  any
	}{
		{`up`, map[string]any{"vectorSelector": map[string]any{"name": "up"}}},
		{
			`http_requests_total{job="api"} offset 5m`,
			map[string]any{"vectorSelector": map[string]any{
				"name":     "http_requests_total",
				"matchers": []any{map[string]any{"name": "job", "type": "=", "value": "api"}},
				"offset":   "5m",
			}},
		},
		{
			`rate(http_requests_total[5m])`,
			map[string]any{"call": map[string]any{
				"func": "rate",
				"args": []any{map[string]any{"matrixSelector": map[string]any{"name": "http_requests_total", "range": "5m"}}},
			}},
		},
		{`-up`, map[string]any{"neg": map[string]any{"vectorSelector": map[string]any{"name": "up"}}}},
		{`+up`, map[string]any{"pos": map[string]any{"vectorSelector": map[string]any{"name": "up"}}}},
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			got, err := Decode(tc.query)
			if err != nil {
				t.Fatalf("Decode(%q): %v", tc.query, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Decode(%q) =\n %#v\nwant\n %#v", tc.query, got, tc.want)
			}
		})
	}
}

func TestDecodeInvalid(t *testing.T) {
	for _, q := range invalidQueries {
		if _, err := Decode(q); err == nil {
			t.Errorf("Decode(%q) = nil error, want error", q)
		}
	}
}

// TestOpMapsAreBijective guards against a future edit that adds an operator
// whose parser.ItemType collides with an existing one: invertOps would silently
// drop it, and decode would then fail to name that operator. The inverted map
// must have exactly as many entries as the forward map.
func TestOpMapsAreBijective(t *testing.T) {
	if len(binaryOpNames) != len(binaryOps) {
		t.Errorf("binaryOps has colliding ItemType values: %d forward, %d inverted", len(binaryOps), len(binaryOpNames))
	}
	if len(aggOpNames) != len(aggOps) {
		t.Errorf("aggOps has colliding ItemType values: %d forward, %d inverted", len(aggOps), len(aggOpNames))
	}
}
