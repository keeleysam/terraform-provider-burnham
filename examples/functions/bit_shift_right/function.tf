// Right shift by 3 bit positions (integer divide by 2^3).
output "divided" {
  value = provider::burnham::bit_shift_right(1024, 3)
  // → 128
}

// Arithmetic shift floors a negative value toward negative infinity.
output "negative" {
  value = provider::burnham::bit_shift_right(-8, 1)
  // → -4
}
