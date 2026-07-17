// Left shift by 10 bit positions (multiply by 2^10).
output "kib" {
  value = provider::burnham::bit_shift_left(1, 10)
  // → 1024
}

// Arbitrary precision: shifting past 64 bits is exact.
output "two_to_the_hundred" {
  value = provider::burnham::bit_shift_left(1, 100)
  // → 1267650600228229401496703205376
}
