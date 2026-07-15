// oelencode: build an Okta Expression Language string from an HCL data tree (no
// string templating, no manual quote escaping). References are marked with
// `ident`; everything else (strings, numbers, lists) is a literal.

// A group-rule expression, assembled structurally instead of as an escaped string.
output "group_rule" {
  value = provider::burnham::oelencode({
    and = [
      { call = {
        class  = "String"
        method = "stringContains"
        args   = [{ ident = "user.department" }, "Engineering"]
      } },
      { "!" = { ident = "user.isContractor" } },
    ]
  })
  // → String.stringContains(user.department, "Engineering") AND !user.isContractor
}

// Assembled from Terraform data with a `for` expression: membership in any of a
// dynamic set of departments.
output "region_gate" {
  value = provider::burnham::oelencode({
    or = [for d in ["Sales", "Marketing"] : { "==" = [{ ident = "user.department" }, d] }]
  })
  // → user.department=="Sales" OR user.department=="Marketing"
}

// A bare group-membership builtin.
output "any_admin_group" {
  value = provider::burnham::oelencode({
    call = {
      function = "isMemberOfAnyGroup"
      args     = ["00gb4o8b4kFEKqzMI0h7", "00gb4o8b4kFEKqzMI0h8"]
    }
  })
  // → isMemberOfAnyGroup("00gb4o8b4kFEKqzMI0h7", "00gb4o8b4kFEKqzMI0h8")
}

// A profile-mapping transform: a ternary that picks a value from an attribute.
output "profile_mapping" {
  value = provider::burnham::oelencode({
    cond = [
      { "==" = [{ ident = "user.groupCode" }, 123] },
      "Sales",
      "Other",
    ]
  })
  // → user.groupCode==123 ? "Sales" : "Other"
}

// A receiver method call: Okta's recommended group-rule status check.
output "internal_status" {
  value = provider::burnham::oelencode({
    "==" = [
      { call = { target = { ident = "user" }, method = "getInternalProperty", args = ["status"] } },
      "ACTIVE",
    ]
  })
  // → user.getInternalProperty("status")=="ACTIVE"
}

// An Identity Governance scope: object-argument group membership, built with `map`.
output "iga_scope" {
  value = provider::burnham::oelencode({
    call = {
      target = { ident = "user" }
      method = "isMemberOf"
      args = [{
        map = [
          { key = "group.profile.name", value = "West Coast Users" },
          { key = "operator", value = "EXACT" },
        ]
      }]
    }
  })
  // → user.isMemberOf({"group.profile.name": "West Coast Users", "operator": "EXACT"})
}
