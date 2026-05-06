// Bulk operation: convert an IPv4 allowlist into NAT64 IPv6 ranges, then concat() to get a dual-stack list.
locals {
  ipv4_allow = ["203.0.113.0/24", "198.51.100.0/24"]
  ipv6_allow = provider::burnham::nat64_synthesize_cidrs(local.ipv4_allow, "64:ff9b::/96")
  // → ["64:ff9b::203.0.113.0/120", "64:ff9b::198.51.100.0/120"]
  dual_stack = concat(local.ipv4_allow, local.ipv6_allow)
}
