# Encode a value as HuJSON. Optional comments mirror the data structure.
output "acl_text" {
  value = provider::burnham::hujsonencode(
    { acls = [], groups = {} },
    { comments = { acls = "Network ACL rules", groups = "Group definitions" } },
  )
}
