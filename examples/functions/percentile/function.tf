/*
Linear-interpolation percentile (Hyndman & Fan Type 7) — same definition as numpy.percentile, R's default, and Excel's PERCENTILE.INC.

Use it to set thresholds based on observed distributions, e.g. "scale instance
count to handle the p95 of last week's request rate" without hand-rolling a
sort + index in HCL.
*/
locals {
  request_rates = [120, 145, 132, 158, 199, 180, 175, 165, 162, 211]
}

output "p50" {
  value = provider::burnham::percentile(local.request_rates, 50)
}

output "p95" {
  value = provider::burnham::percentile(local.request_rates, 95)
  // h = 0.95 * 9 = 8.55 → between sorted[8]=199 and sorted[9]=211 → 199 + 0.55*12 = 205.6
}

output "p100_is_max" {
  value = provider::burnham::percentile(local.request_rates, 100)
  // → 211
}
