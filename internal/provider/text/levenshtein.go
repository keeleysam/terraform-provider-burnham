/*
Levenshtein edit distance: the minimum number of single-character edits (insertions, deletions, or substitutions) required to turn one string into another.

Useful for "did-you-mean?" suggestions and detecting near-duplicate names. We implement the algorithm directly over Unicode codepoints (not bytes), so `levenshtein("café", "cafe") == 1` regardless of whether the strings are NFC or NFD encoded: the rune count is what matters. For mixed normalization, run `unicode_normalize` first.

The implementation is the classic two-row dynamic-programming variant: O(min(n, m)) space, O(n·m) time.
*/

package text

import (
	"context"
	_ "embed"
	"fmt"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

// levenshteinMaxBytes caps each input's byte length. This bounds the rune conversion and DP allocations; it is a sanity guard on raw input size, not the latency bound. 256 KiB is far above any realistic identifier or name (typical inputs are < 100 chars).
const levenshteinMaxBytes = 256 * 1024

// levenshteinMaxProduct caps the number of DP cells, i.e. runes(a) * runes(b). The DP does O(n·m) work, so the product of the two rune counts (not either length alone) is what bounds latency. Two 256 KiB inputs would be ~6.9e10 cells and take ~110s; capping the product near 2e9 holds the adversarial worst case to a few seconds while still admitting any realistic pairing (identifiers, names, even paragraphs of prose sit orders of magnitude below it). Pairings past the cap return an argument error instead of blocking plan-time evaluation.
const levenshteinMaxProduct = 2_000_000_000

var _ function.Function = (*LevenshteinFunction)(nil)

//go:embed descriptions/levenshtein.md
var levenshteinDescription string

type LevenshteinFunction struct{}

func NewLevenshteinFunction() function.Function { return &LevenshteinFunction{} }

func (f *LevenshteinFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "levenshtein"
}

func (f *LevenshteinFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Levenshtein edit distance between two strings",
		MarkdownDescription: levenshteinDescription,
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
	if len(a) > levenshteinMaxBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("a exceeds maximum supported length of %d bytes", levenshteinMaxBytes))
		return
	}
	if len(b) > levenshteinMaxBytes {
		resp.Error = function.NewArgumentFuncError(1, fmt.Sprintf("b exceeds maximum supported length of %d bytes", levenshteinMaxBytes))
		return
	}
	// The DP is O(runes(a)·runes(b)), so bound the product, not the individual lengths: a short string paired with a long one is cheap, but two large strings are not. Rejecting here keeps the worst case to a few seconds.
	if int64(utf8.RuneCountInString(a))*int64(utf8.RuneCountInString(b)) > levenshteinMaxProduct {
		resp.Error = function.NewFuncError(fmt.Sprintf("inputs too large: the edit-distance matrix would exceed %d cells; reduce the length of one or both arguments", levenshteinMaxProduct))
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
