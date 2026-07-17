// Combine flag bits into a single value.
output "flags" {
  value = provider::burnham::bit_or([1, 2, 8])
  // → 11
}
