// Bound a number to a closed interval. Equivalent to max(min_val, min(max_val, value)) but reads cleanly.
output "in_range" {
  value = provider::burnham::clamp(5, 0, 10)
  // → 5
}

output "below_min" {
  value = provider::burnham::clamp(-3, 0, 10)
  // → 0
}

output "above_max" {
  value = provider::burnham::clamp(99, 0, 10)
  // → 10
}
