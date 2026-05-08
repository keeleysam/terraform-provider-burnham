// Package text provides text manipulation and rendering provider-defined functions: Unicode normalisation, Unicode-aware slugification, Levenshtein edit distance, word-wrap, cowsay, and ASCII QR codes.
package text

import "github.com/hashicorp/terraform-plugin-framework/function"

// textMaxInputBytes caps input length for the text-family transformations (`slugify`, `unicode_normalize`, `wrap`). The underlying libraries (gosimple/slug, golang.org/x/text/unicode/norm, mitchellh/go-wordwrap) do not enforce internal limits and walk the input rune-by-rune, so a multi-MB string would block plan-time evaluation. 4 MiB is several orders of magnitude above any realistic identifier, slug, or wrappable text. Levenshtein and cowsay have their own tighter caps (256 KiB and 64 KiB respectively) reflecting their O(n·m) and visual-output natures.
const textMaxInputBytes = 4 * 1024 * 1024

// Functions returns the text provider-defined functions registered by terraform-burnham.
func Functions() []func() function.Function {
	return []func() function.Function{
		NewUnicodeNormalizeFunction,
		NewSlugifyFunction,
		NewLevenshteinFunction,
		NewWrapFunction,
		NewCowsayFunction,
		NewQRAsciiFunction,
	}
}
