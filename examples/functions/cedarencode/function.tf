// cedarencode: build a Cedar policy (DSL) from its EST (the Cedar JSON policy
// format) data tree. cedarencode(cedardecode(x)) round-trips.
output "policy" {
  value = provider::burnham::cedarencode({
    effect     = "permit"
    principal  = { op = "==", entity = { type = "User", id = "alice" } }
    action     = { op = "==", entity = { type = "Action", id = "view" } }
    resource   = { op = "in", entity = { type = "Album", id = "vacation" } }
    conditions = []
  })
  /* →
  permit (
      principal == User::"alice",
      action == Action::"view",
      resource in Album::"vacation"
  );
  */
}

// A `when` condition is an EST expression tree. The easiest way to discover its
// shape is to write the policy as text and run it through cedardecode.
output "conditional_policy" {
  value = provider::burnham::cedarencode({
    effect    = "permit"
    principal = { op = "All" }
    action    = { op = "==", entity = { type = "Action", id = "editPhoto" } }
    resource  = { op = "All" }
    conditions = [{
      kind = "when"
      body = { "==" = {
        left  = { "." = { left = { Var = "resource" }, attr = "owner" } }
        right = { Var = "principal" }
      } }
    }]
  })
  /* →
  permit (
      principal,
      action == Action::"editPhoto",
      resource
  )
  when { resource.owner == principal };
  */
}
