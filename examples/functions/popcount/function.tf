// Number of set bits (Hamming weight).
output "all_ones_byte" {
  value = provider::burnham::popcount(255)
  // → 8
}

output "none" {
  value = provider::burnham::popcount(0)
  // → 0
}
