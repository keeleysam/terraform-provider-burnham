/*
Levenshtein edit distance — the minimum number of single-character edits (insertions, deletions, or substitutions) required to turn one string into another.

Useful for "did-you-mean?" suggestions and detecting near-duplicate names. We implement the algorithm directly over Unicode codepoints (not bytes), so `levenshtein("café", "cafe") == 1` regardless of whether the strings are NFC or NFD encoded — the rune count is what matters. For mixed normalization, run `unicode_normalize` first.

The implementation is the classic two-row dynamic-programming variant — O(min(n, m)) space, O(n·m) time.
*/

package text

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

var _ function.Function = (*LevenshteinFunction)(nil)

type LevenshteinFunction struct{}

func NewLevenshteinFunction() function.Function { return &LevenshteinFunction{} }

func (f *LevenshteinFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "levenshtein"
}

func (f *LevenshteinFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Levenshtein edit distance between two strings",
		MarkdownDescription: "Returns the [Levenshtein distance](https://en.wikipedia.org/wiki/Levenshtein_distance) between `a` and `b` — the minimum number of single-character insertions, deletions, or substitutions needed to turn one string into the other.\n\nDistance is computed over Unicode codepoints, not bytes — so `levenshtein(\"café\", \"cafe\")` is `1` regardless of byte length. If your inputs may be in different normalization forms (NFC vs NFD), run `unicode_normalize(s, \"NFC\")` first.\n\nClassic uses: \"did-you-mean\" suggestions in dynamic config selection (`closest_match` over a list), spotting typos in resource names, deduplicating near-identical entries.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "a", Description: "First string."},
			function.StringParameter{Name: "b", Description: "Second string."},
		},
		Return: function.Int64Return{},
	}
}

func (f *LevenshteinFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var a, b string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &a, &b))
	if resp.Error != nil {
		return
	}
	d := int64(levenshteinDistance(a, b))
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &d))
}

// levenshteinDistance computes the edit distance between two strings, working in Unicode codepoints. Two-row DP: O(min(n, m)) space, O(n·m) time.
func levenshteinDistance(a, b string) int {
	ar := []rune(a)
	br := []rune(b)
	// Make ar the shorter to keep the inner row small.
	if len(ar) > len(br) {
		ar, br = br, ar
	}
	if len(ar) == 0 {
		return len(br)
	}

	prev := make([]int, len(ar)+1)
	curr := make([]int, len(ar)+1)
	for i := range prev {
		prev[i] = i
	}

	for j := 1; j <= len(br); j++ {
		curr[0] = j
		for i := 1; i <= len(ar); i++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			curr[i] = min(
				curr[i-1]+1,    // insertion
				prev[i]+1,      // deletion
				prev[i-1]+cost, // substitution
			)
		}
		prev, curr = curr, prev
	}
	return prev[len(ar)]
}
