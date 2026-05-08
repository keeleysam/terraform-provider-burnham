// Plain arithmetic mean of a list of numbers.
output "uptime_seconds_mean" {
  value = provider::burnham::mean([0.42, 0.51, 0.39, 0.55, 0.47])
  // → 0.468
}
