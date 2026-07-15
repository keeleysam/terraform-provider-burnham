package promql

import "testing"

// TestMarshalRoundTrip drives the full Terraform-value boundary that the
// promqldecode -> promqlencode chain crosses at runtime: Decode produces a Go
// tree, nodeToAttr marshals it into attr.Values (the promqldecode return),
// terraformToNode reads those back (the promqlencode argument), and Encode
// re-serializes. The contract is the same as the pure-Go round trip, but this
// exercises nodeToAttr (TupleValue / ObjectValue / NumberValue construction) and
// terraformToNode, which the Go-tree tests bypass.
func TestMarshalRoundTrip(t *testing.T) {
	queries := []string{
		`up`,
		`http_requests_total{code=~"5..", job="api"}`,
		`label_replace(up{job="api"}, "service", "$1", "job", "(.*)")`, // heterogeneous tuple: object mixed with bare strings
		`time()`,          // call with an empty args tuple
		`up @ 1609746000`, // large integral timestamp through big.Float
		`rate(http_requests_total[5m])[10m:1m] @ 100.5`, // fractional @ on a subquery
		`sum by (job) (rate(errors_total[5m])) / on (job) group_left (env) sum by (job) (rate(requests_total[5m]))`,
		`1.5`,
		`"a string"`,
	}
	for _, q := range queries {
		t.Run(q, func(t *testing.T) {
			want, err := Format(q, false)
			if err != nil {
				t.Fatalf("Format(%q): %v", q, err)
			}
			tree, err := Decode(q)
			if err != nil {
				t.Fatalf("Decode(%q): %v", q, err)
			}
			av, err := nodeToAttr(tree)
			if err != nil {
				t.Fatalf("nodeToAttr: %v", err)
			}
			back, err := terraformToNode(av)
			if err != nil {
				t.Fatalf("terraformToNode: %v", err)
			}
			got, err := Encode(back)
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}
			if got != want {
				t.Fatalf("marshal round trip = %q, want %q", got, want)
			}
		})
	}
}
