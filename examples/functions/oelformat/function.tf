// oelformat: canonicalize a hand-written Okta EL string, normalizing spacing
// and quoting. Fails the plan on invalid input (use oelvalidate for a
// non-failing check).
output "canonical" {
  value = provider::burnham::oelformat("user.department  ==  'Sales'")
  // → user.department=="Sales"
}

// Two expressions differing only in spacing/quoting canonicalize identically.
output "normalized_call" {
  value = provider::burnham::oelformat("String.stringContains( user.department , 'Sales' )")
  // → String.stringContains(user.department, "Sales")
}
