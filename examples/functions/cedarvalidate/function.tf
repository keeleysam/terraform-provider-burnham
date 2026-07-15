// cedarvalidate: is a string a syntactically valid Cedar policy document?
// Returns a bool and never fails the plan, so it suits a precondition.
output "valid" {
  value = provider::burnham::cedarvalidate("permit (principal, action, resource) when { resource.owner == principal };")
  // → true
}

output "invalid" {
  value = provider::burnham::cedarvalidate("permit (principal, action, resource")
  // → false
}
