// cedardecode: parse a Cedar policy (DSL) into its EST data tree, for
// inspecting or patching a policy as structured data. Inverse of cedarencode.
output "est" {
  value = provider::burnham::cedardecode("permit (principal == User::\"alice\", action == Action::\"view\", resource);")
  /* → {
    effect    = "permit"
    principal = { op = "==", entity = { type = "User", id = "alice" } }
    action    = { op = "==", entity = { type = "Action", id = "view" } }
    resource  = { op = "All" }
  }
  */
}
