// pi_digits — RFC 3091 §1 TCP service for π. Returns the first `count` digits following the decimal point. The leading 3 of π = 3.1415… is implied per RFC and never emitted.
output "first_ten_digits" {
  value = provider::burnham::pi_digits(10)
  // → "1415926535"
}

output "first_thirty_digits" {
  value = provider::burnham::pi_digits(30)
  // → "141592653589793238462643383279"
}
