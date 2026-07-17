// Bitwise AND folded over a list. Terraform has no bitwise operators.
output "masked" {
  value = provider::burnham::bit_and([12, 10])
  // → 8
}

// Keep only the low 8 bits of a value.
output "low_byte" {
  value = provider::burnham::bit_and([511, 255])
  // → 255
}
