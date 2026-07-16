// urldecode: percent-decode (a function Terraform core lacks).
// mode controls "+": space in "query" (default), literal in "path".
output "value" {
  value = provider::burnham::urldecode("a+b%2Fc")
  // → "a b/c"
}
