// Validate that a user-supplied IP falls within an expected subnet.
output "in_mgmt_subnet" {
  value = provider::burnham::ip_in_cidr("10.0.1.50", "10.0.1.0/24")
  // → true
}
