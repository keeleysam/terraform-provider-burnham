// celevaluate: evaluate a standard CEL expression at plan time against variable bindings.
// Handy for testing the logic of an expression built with celencode.
output "allowed" {
  value = provider::burnham::celevaluate(
    "request.tier == \"prod\" && \"admin\" in request.roles",
    { vars = { request = { tier = "prod", roles = ["viewer", "admin"] } } },
  )
  // → true
}

// Compute a value; result types map to Terraform (timestamp → RFC3339 string by default).
output "cutoff" {
  value = provider::burnham::celevaluate(
    "timestamp(start) + duration(\"720h\")",
    { vars = { start = "2026-01-01T00:00:00Z" } },
  )
  // → "2026-01-31T00:00:00Z"
}
