// Parse an arbitrary HCL document into a value. Static literals only — references and function calls have no eval context. Block syntax is not supported.
output "config" {
  value = provider::burnham::hcldecode("name = \"web\"\nreplicas = 3\ntags = [\"prod\", \"east\"]")
}
/* →
{
  name     = "web"
  replicas = 3
  tags     = ["prod", "east"]
}
*/
