// Usable host count: subtracts network + broadcast for IPv4. RFC-correct edge cases: /31 = 2 (point-to-point), /32 = 1.
output "usable" {
  value = provider::burnham::cidr_usable_host_count("10.0.0.0/24")
  // → 254
}
