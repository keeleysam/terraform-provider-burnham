// Bitwise XOR folded over a list.
output "toggled" {
  value = provider::burnham::bit_xor([5, 3])
  // → 6
}

// XOR is its own inverse, so folding a value in twice cancels it.
output "cancelled" {
  value = provider::burnham::bit_xor([5, 3, 6])
  // → 0
}
