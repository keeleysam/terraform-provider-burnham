// Package text provides text manipulation and rendering provider-defined functions: Unicode normalisation, Unicode-aware slugification, Levenshtein edit distance, word-wrap, cowsay, and ASCII QR codes.
package text

import "github.com/hashicorp/terraform-plugin-framework/function"

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
