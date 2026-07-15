// oelvalidate: is a string a syntactically valid Okta EL expression? Returns a
// bool and never fails the plan, so it suits a precondition guarding a
// hand-written expression.
output "valid" {
  value = provider::burnham::oelvalidate("String.stringContains(user.department, \"Sales\")")
  // → true
}

output "invalid" {
  value = provider::burnham::oelvalidate("String.stringContains(user.department,")
  // → false
}
