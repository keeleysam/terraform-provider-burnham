// oelevaluate: evaluate an Okta EL expression against a sample context, to
// preview or test a group rule at plan time. It is a local approximation of
// the group-rule subset, not Okta's server-side engine.
output "sales_rule" {
  value = provider::burnham::oelevaluate("user.department == \"Sales\"", {
    user = { department = "Sales" }
  })
  // → true
}

// Group membership is supplied via group_ids (and groups, for name lookups).
output "membership" {
  value = provider::burnham::oelevaluate("isMemberOfGroupName(\"Engineering\")", {
    group_ids = ["00g1"]
    groups    = { "00g1" = { profile = { name = "Engineering" } } }
  })
  // → true
}
