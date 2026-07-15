// oeldecode: parse an Okta EL string into the oelencode data tree (the inverse
// of oelencode), for testing or migrating hand-written expressions into the
// data model.
output "tree" {
  value = provider::burnham::oeldecode("user.department == \"Sales\"")
  // → { "==" = [{ ident = "user.department" }, "Sales"] }
}

// oelencode(oeldecode(x)) round-trips to the canonical form of x.
output "roundtrip" {
  value = provider::burnham::oelencode(provider::burnham::oeldecode("String.stringContains( user.dept , 'x' )"))
  // → String.stringContains(user.dept, "x")
}
