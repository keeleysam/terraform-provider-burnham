// Single IPv4 CIDR → NAT64 IPv6 CIDR. /64 and /96 prefixes only.
output "nat64_pool" {
  value = provider::burnham::nat64_synthesize_cidr("192.0.2.0/24", "64:ff9b::/96")
  // → "64:ff9b::192.0.2.0/120"
}
