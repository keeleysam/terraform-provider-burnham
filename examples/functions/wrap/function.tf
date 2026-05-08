// Word-wrap to a column width. Existing newlines are preserved; words longer than the width are not split (they overflow on their own line).
output "wrapped_at_20" {
  value = provider::burnham::wrap("The quick brown fox jumped over the lazy dog.", 20)
  // → "The quick brown fox\njumped over the lazy\ndog."
}

output "fits_on_one_line" {
  value = provider::burnham::wrap("hello world", 80)
  // → "hello world"
}
