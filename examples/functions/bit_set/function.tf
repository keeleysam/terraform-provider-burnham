// Set bit 3 (place value 2^3 = 8).
output "flag_on" {
  value = provider::burnham::bit_set(0, 3)
  // → 8
}
