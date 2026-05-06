// 4 or 6 for a CIDR.
output "family" {
  value = provider::burnham::cidr_version("10.0.0.0/8")
  // → 4
}
