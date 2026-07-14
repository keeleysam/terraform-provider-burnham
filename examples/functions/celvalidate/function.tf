// celvalidate: true if the string is a syntactically valid CEL expression.
// Unlike celformat, invalid input returns false instead of failing the plan.
output "ok" {
  value = provider::burnham::celvalidate("resource.name.startsWith('prod-') && x.exists(i, i > 0)")
  // → true
}

output "bad" {
  value = provider::burnham::celvalidate("a &&")
  // → false
}
