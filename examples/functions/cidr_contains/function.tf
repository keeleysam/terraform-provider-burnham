// Does cidr fully contain other? `other` may be an IP or a CIDR.
output "spoke_in_hub" {
  value = provider::burnham::cidr_contains("10.0.0.0/8", "10.1.2.0/24")
  // → true
}
