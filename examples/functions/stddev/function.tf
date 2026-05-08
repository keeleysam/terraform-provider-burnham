// Population standard deviation, σ = √(variance).
output "stddev" {
  value = provider::burnham::stddev([2, 4, 4, 4, 5, 5, 7, 9])
  // → 2  (population variance is 4)
}
