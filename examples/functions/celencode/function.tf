// celencode: build a CEL expression from an HCL data tree (no string templating).
// References are marked with `ident`; everything else (strings, numbers, lists) is a literal.
output "iam_condition" {
  value = provider::burnham::celencode({
    "&&" = [
      { "==" = [{ ident = "device.os_type" }, { ident = "OsType.DESKTOP_MAC" }] },
      { "in" = [{ ident = "origin.region_code" }, ["US", "CA", "GB"]] },
      { call = {
        target   = { ident = "resource.name" }
        function = "startsWith"
        args     = ["projects/_/buckets/prod-"]
      } },
    ]
  })
  // → device.os_type == OsType.DESKTOP_MAC && origin.region_code in ["US", "CA", "GB"] && resource.name.startsWith("projects/_/buckets/prod-")
}

// Assembled from Terraform data with a `for` expression, still no CEL string concatenation.
output "region_gate" {
  value = provider::burnham::celencode({
    or = [for r in ["US", "CA"] : { "==" = [{ ident = "origin.region_code" }, r] }]
  })
  // → origin.region_code == "US" || origin.region_code == "CA"
}

// A macro (has/all/exists/exists_one/map/filter) is a call whose bound variable
// is passed as an `ident` argument.
output "any_admin_group" {
  value = provider::burnham::celencode({
    call = {
      target   = { ident = "user.groups" }
      function = "exists"
      args = [
        { ident = "g" },
        { call = { target = { ident = "g" }, function = "startsWith", args = ["admin-"] } },
      ]
    }
  })
  // → user.groups.exists(g, g.startsWith("admin-"))
}

// The same expression in the canonical notation (syntax.proto field names,
// operators as calls). Both notations are accepted and may be mixed.
output "canonical" {
  value = provider::burnham::celencode({
    call_expr = {
      function = "_==_"
      args = [
        { select_expr = { operand = { ident_expr = { name = "device" } }, field = "os_type" } },
        { ident_expr = { name = "OsType.DESKTOP_MAC" } },
      ]
    }
  })
  // → device.os_type == OsType.DESKTOP_MAC
}
