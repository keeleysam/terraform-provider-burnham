// Edit distance between two strings, counted in Unicode codepoints. Useful for "did-you-mean" suggestions and detecting near-duplicates.
output "kitten_to_sitting" {
  value = provider::burnham::levenshtein("kitten", "sitting")
  // → 3   (the classic textbook example: substitution + substitution + insertion)
}

output "identical_is_zero" {
  value = provider::burnham::levenshtein("hello", "hello")
  // → 0
}

output "unicode_counts_codepoints" {
  value = provider::burnham::levenshtein("café", "cafe")
  // → 1   (one codepoint differs, regardless of UTF-8 byte length)
}
