/*
Least common multiple of a list of integers: the smallest non-negative integer every element divides.
Any list containing a zero has an lcm of 0.
*/
output "two_values" {
  value = provider::burnham::lcm([4, 6])
  // → 12
}

output "three_values" {
  value = provider::burnham::lcm([2, 3, 4])
  // → 12
}

output "zero_present" {
  value = provider::burnham::lcm([0, 5])
  // → 0
}

output "single_value" {
  value = provider::burnham::lcm([7])
  // → 7
}
