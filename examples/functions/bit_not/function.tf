// Complement within an 8-bit field: every bit of the byte flips.
output "inverted_byte" {
  value = provider::burnham::bit_not(0, 8)
  // → 255
}

// Complement within a 4-bit nibble.
output "inverted_nibble" {
  value = provider::burnham::bit_not(1, 4)
  // → 14
}
