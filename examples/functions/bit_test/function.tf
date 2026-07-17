// Report whether a single bit is set.
output "bit_three_set" {
  value = provider::burnham::bit_test(8, 3)
  // → true
}

output "bit_zero_set" {
  value = provider::burnham::bit_test(8, 0)
  // → false
}
