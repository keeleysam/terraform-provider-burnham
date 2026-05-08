// Mode is always returned as a list because the data may be multimodal.
output "unimodal" {
  value = provider::burnham::mode([1, 2, 2, 3])
  // → [2]
}

output "bimodal" {
  value = provider::burnham::mode([1, 1, 2, 2, 3])
  // → [1, 2]
}

output "all_unique_no_mode" {
  value = provider::burnham::mode([3, 1, 2])
  // → []   (no value repeats; mode is undefined)
}

output "single_element" {
  value = provider::burnham::mode([5])
  // → [5]   (degenerate one-element case: the value is trivially the mode)
}
