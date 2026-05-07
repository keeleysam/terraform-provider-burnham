// pi_approximate_digit — RFC 3091 §2.2 UDP reply for 22/7. Long division of 22 by 7 gives 3.142857142857…, a period-6 cycle of "142857".
output "first_digit" {
  value = provider::burnham::pi_approximate_digit(1)
  // → "1:1"
}

output "cycle_wraps_at_seven" {
  value = provider::burnham::pi_approximate_digit(7)
  // → "7:1"   (back to start of "142857")
}

// 22/7 has no upper bound — even astronomical n returns instantly.
output "millionth_digit" {
  value = provider::burnham::pi_approximate_digit(1000000)
  // → "1000000:8"
}
