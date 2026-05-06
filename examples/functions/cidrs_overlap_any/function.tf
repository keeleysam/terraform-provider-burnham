# Bulk conflict check: does any CIDR in `a` overlap any in `b`?
output "any_collision" {
  value = provider::burnham::cidrs_overlap_any(
    ["10.4.0.0/16", "10.5.0.0/16"],
    ["10.0.0.0/16", "10.4.0.0/16"],
  )
}
