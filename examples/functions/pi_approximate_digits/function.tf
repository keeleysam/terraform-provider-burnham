// pi_approximate_digits — RFC 3091 §1.1 TCP approximate service for 22/7. Returns the first `count` digits of 22/7 = 3.142857142857… (period-6).
output "two_cycles" {
  value = provider::burnham::pi_approximate_digits(12)
  // → "142857142857"
}

output "partial_cycle" {
  value = provider::burnham::pi_approximate_digits(5)
  // → "14285"
}
