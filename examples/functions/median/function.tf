// Median of an odd-length list is the middle value of the sorted list.
output "odd_count" {
  value = provider::burnham::median([5, 1, 3, 2, 4])
  // → 3
}

// For an even-length list, the median is the mean of the two central values.
output "even_count" {
  value = provider::burnham::median([1, 2, 3, 4])
  // → 2.5
}
