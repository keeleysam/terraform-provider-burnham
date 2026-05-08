// Mode is always returned as a list because the data may be multimodal.
output "unimodal" {
  value = provider::burnham::mode([1, 2, 2, 3])
  // → [2]
}

output "bimodal" {
  value = provider::burnham::mode([1, 1, 2, 2, 3])
  // → [1, 2]
}

output "all_unique_each_appears_once" {
  value = provider::burnham::mode([3, 1, 2])
  // → [1, 2, 3]   (every value is a mode; sorted ascending)
}
