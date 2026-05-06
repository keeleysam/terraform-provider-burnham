// Last address in a CIDR (broadcast for IPv4).
output "subnet_top" {
  value = provider::burnham::cidr_last_ip("10.0.0.0/24")
  // → "10.0.0.255"
}
