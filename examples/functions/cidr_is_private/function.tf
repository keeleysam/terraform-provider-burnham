// Same coverage as ip_is_private, applied to a whole CIDR.
output "internal_block" {
  value = provider::burnham::cidr_is_private("10.0.0.0/8")
  // → true
}
