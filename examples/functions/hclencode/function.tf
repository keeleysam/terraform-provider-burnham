// Encode an object as HCL attribute statements (one per member, alphabetical order).
output "hcl" {
  value = provider::burnham::hclencode({
    name     = "web"
    replicas = 3
    tags     = ["prod", "east"]
  })
}
/* →
name     = "web"
replicas = 3
tags     = ["prod", "east"]
*/
