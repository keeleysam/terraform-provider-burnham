// Address space common to two lists.
output "vpn_overlap" {
  value = provider::burnham::cidr_intersect(
    ["10.0.0.0/8", "172.16.0.0/12"],
    ["10.100.0.0/16", "192.168.0.0/16"],
  )
  // → ["10.100.0.0/16"]
}
