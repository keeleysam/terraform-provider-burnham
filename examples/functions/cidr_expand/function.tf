# All individual IPs in a small CIDR. Capped at 65536 addresses.
output "host_ips" {
  value = provider::burnham::cidr_expand("10.0.0.0/30")
  # → ["10.0.0.0", "10.0.0.1", "10.0.0.2", "10.0.0.3"]
}
