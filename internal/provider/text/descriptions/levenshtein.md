<!-- Edit here: this is the MarkdownDescription source for the burnham levenshtein function. docs/functions/levenshtein.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns the [Levenshtein distance](https://en.wikipedia.org/wiki/Levenshtein_distance) between `a` and `b`: the minimum number of single-character insertions, deletions, or substitutions needed to turn one string into the other.

Distance is computed over Unicode codepoints, not bytes, so `levenshtein("café", "cafe")` is `1` regardless of byte length. If your inputs may be in different normalization forms (NFC vs NFD), run `unicode_normalize(s, "NFC")` first.

Classic uses:

- "Did-you-mean" suggestions: pick the closest valid option from a list by mapping each candidate to its distance from the input and taking the minimum.
- Spotting typos in resource names.
- Deduplicating near-identical entries.

The underlying dynamic-programming algorithm is O(n·m), so latency is bounded by the product of the two rune counts, not either length alone. Realistic inputs (identifiers, resource names, even paragraphs of prose) sit comfortably below the limits.

~> **Note:** Each input is capped at 256 KiB, and the number of matrix cells (`runes(a) × runes(b)`) is capped so the worst case stays within a few seconds. A pairing that would exceed the cap returns an error instead of blocking plan-time evaluation.