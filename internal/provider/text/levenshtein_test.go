package text

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runLevenshtein(t *testing.T, a, b string) (int64, *function.FuncError) {
	t.Helper()
	f := &LevenshteinFunction{}
	args := function.NewArgumentsData([]attr.Value{types.StringValue(a), types.StringValue(b)})
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.Int64Value(0))}
	f.Run(context.Background(), req, resp)
	if resp.Error != nil {
		return 0, resp.Error
	}
	result, ok := resp.Result.Value().(types.Int64)
	if !ok {
		t.Fatalf("expected Int64 result, got %T", resp.Result.Value())
	}
	return result.ValueInt64(), nil
}

func TestLevenshtein_Basic(t *testing.T) {
	cases := []struct {
		a, b string
		want int64
	}{
		{"", "", 0},
		{"abc", "abc", 0},
		{"kitten", "sitting", 3},
		{"café", "cafe", 1},
		{"abc", "", 3},
	}
	for _, c := range cases {
		got, err := runLevenshtein(t, c.a, c.b)
		if err != nil {
			t.Errorf("levenshtein(%q, %q) errored: %s", c.a, c.b, err.Text)
			continue
		}
		if got != c.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

// TestLevenshtein_RejectsBeyondProductCap guards the worst-case latency bound: a pairing whose DP matrix (runes(a)*runes(b)) exceeds the cap must be rejected quickly rather than running the full O(n*m) DP (which for two large inputs is tens of seconds). Both inputs sit under the per-string byte cap, so only the product bound can reject them.
func TestLevenshtein_RejectsBeyondProductCap(t *testing.T) {
	n := 50_000 // 50000 * 50000 = 2.5e9 cells, above levenshteinMaxProduct.
	a := strings.Repeat("a", n)
	b := strings.Repeat("b", n)
	start := time.Now()
	_, err := runLevenshtein(t, a, b)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatalf("levenshtein over two %d-rune inputs (%d cells) should error", n, int64(n)*int64(n))
	}
	if elapsed > 2*time.Second {
		t.Fatalf("rejection took %v, expected a fast bail-out (the whole point of the cap)", elapsed)
	}
}
