// JSONata validate: check an expression is well-formed without failing the plan.
output "is_valid" {
  value = provider::burnham::jsonata_validate("orders[price > 10].product")
  // → true
}

output "is_invalid" {
  value = provider::burnham::jsonata_validate("orders[price > ")
  // → false
}
