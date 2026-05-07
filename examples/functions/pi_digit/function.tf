// pi_digit — RFC 3091 §2.1.2 UDP reply for π. Returns "<n>:<digit>".
output "first_digit" {
  value = provider::burnham::pi_digit(1)
  // → "1:1"   (π = 3.[1]415…)
}

output "hundredth_digit" {
  value = provider::burnham::pi_digit(100)
  // → "100:9"
}

// The Feynman point: digits 762..767 of π are "999999".
output "feynman_point_first" {
  value = provider::burnham::pi_digit(762)
  // → "762:9"
}
