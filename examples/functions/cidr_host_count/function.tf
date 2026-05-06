// Total addresses in a CIDR. Capped at MaxInt64 for huge IPv6 prefixes.
output "size" {
  value = provider::burnham::cidr_host_count("10.0.0.0/24")
  // → 256
}
