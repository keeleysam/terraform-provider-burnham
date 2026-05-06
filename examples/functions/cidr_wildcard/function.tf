// Cisco-style wildcard mask (bitwise inverse of the subnet mask). IPv4 only.
output "acl_mask" {
  value = provider::burnham::cidr_wildcard("10.0.0.0/24")
  // → "0.0.0.255"
}
