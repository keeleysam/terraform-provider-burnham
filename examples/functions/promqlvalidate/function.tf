// promqlvalidate: is a string a valid PromQL expression? Returns a bool and
// never fails the plan, so it suits a precondition guarding a hand-written query.
output "valid" {
  value = provider::burnham::promqlvalidate("rate(http_requests_total[5m])")
  // → true
}

output "invalid" {
  // rate() needs a range vector, not an instant vector (a type error the parser catches).
  value = provider::burnham::promqlvalidate("rate(http_requests_total)")
  // → false
}
