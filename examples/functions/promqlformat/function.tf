// promqlformat: canonicalize a PromQL query (normalize spacing and operators).
// Pass { pretty = true } for the multi-line form. Fails on invalid input.
output "canonical" {
  value = provider::burnham::promqlformat("sum  (  rate( http_requests_total [5m] ) )")
  // → sum(rate(http_requests_total[5m]))
}

// The pretty form wraps only long sub-expressions.
output "pretty" {
  value = provider::burnham::promqlformat(
    "sum by (job) (rate(http_requests_total{code=~\"5..\"}[5m])) / sum by (job) (rate(http_requests_total[5m])) > 0.05",
    { pretty = true }
  )
  /* →
    sum by (job) (rate(http_requests_total{code=~"5.."}[5m]))
  /
    sum by (job) (rate(http_requests_total[5m]))
>
  0.05
  */
}
