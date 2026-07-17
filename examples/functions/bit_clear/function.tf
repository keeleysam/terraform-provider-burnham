// Clear bit 1 (place value 2^1 = 2) of 15.
output "flag_off" {
  value = provider::burnham::bit_clear(15, 1)
  // → 13
}
