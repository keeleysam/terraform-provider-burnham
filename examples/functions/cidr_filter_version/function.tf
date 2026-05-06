// Split a dual-stack list by family.
output "ipv4_only" {
  value = provider::burnham::cidr_filter_version(
    ["10.0.0.0/8", "2001:db8::/32", "fd00::/8"],
    4,
  )
  // → ["10.0.0.0/8"]
}
