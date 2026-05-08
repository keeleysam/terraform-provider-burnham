/*
Floor-modulo: result has the sign of the divisor. Built-in `%` follows Go's truncated-modulo (sign of dividend), which is the standard footgun for negative inputs.

Use this when you want a "wrap into [0, b)" idiom that is safe for any integer i — e.g. picking the i-th element of a list when i may be negative.
*/
output "positive_positive" {
  value = provider::burnham::mod_floor(7, 3)
  // → 1
}

output "negative_dividend_wraps_into_range" {
  value = provider::burnham::mod_floor(-7, 3)
  // → 2  (vs the built-in `-7 % 3` which is -1)
}

output "fractional" {
  value = provider::burnham::mod_floor(5.5, 2)
  // → 1.5
}
