/*
Greatest common divisor of a list of integers: the largest integer dividing every element.
Negatives are reduced to absolute value, so the result is always non-negative.
*/
output "two_values" {
  value = provider::burnham::gcd([12, 18])
  // → 6
}

output "three_values" {
  value = provider::burnham::gcd([12, 18, 30])
  // → 6
}

output "zero_and_n" {
  value = provider::burnham::gcd([0, 5])
  // → 5
}

output "negative_operand" {
  value = provider::burnham::gcd([-12, 18])
  // → 6
}
