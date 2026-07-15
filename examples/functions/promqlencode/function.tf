// promqlencode: build a PromQL query from an HCL data tree that mirrors the
// Prometheus AST. Label values are quoted correctly for you, so there is no
// fragile string interpolation.
output "selector" {
  value = provider::burnham::promqlencode({
    vectorSelector = {
      name = "http_requests_total"
      matchers = [
        { name = "job", type = "=", value = "api" },
        { name = "code", type = "=~", value = "5.." },
      ]
    }
  })
  // → http_requests_total{code=~"5..",job="api"}
}

// A full query composed from data: sum by (job) (rate(...[5m])).
output "rate_by_job" {
  value = provider::burnham::promqlencode({
    aggregation = {
      op = "sum"
      by = ["job"]
      expr = { call = {
        func = "rate"
        args = [{ matrixSelector = { name = "http_requests_total", range = "5m" } }]
      } }
    }
  })
  // → sum by (job) (rate(http_requests_total[5m]))
}
