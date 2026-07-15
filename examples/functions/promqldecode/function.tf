// promqldecode: parse a PromQL query into the promqlencode data tree. Useful for
// testing (round-tripping) and for lifting a hand-written query into the model.
output "roundtrip" {
  value = provider::burnham::promqlencode(provider::burnham::promqldecode(
    "rate( errors_total [5m] )  /  rate(requests_total[5m])"
  ))
  // → rate(errors_total[5m]) / rate(requests_total[5m])
}

// The returned tree mirrors promqlencode's node vocabulary. A bare metric name
// carries its implicit __name__ matcher in the name field, not as a matcher.
output "tree" {
  value = provider::burnham::promqldecode("http_requests_total{job=\"api\"} offset 5m")
  /* →
  {
    vectorSelector = {
      name     = "http_requests_total"
      matchers = [{ name = "job", type = "=", value = "api" }]
      offset   = "5m"
    }
  }
  */
}
