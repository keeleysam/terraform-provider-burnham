# Given an IP, find every matching CIDR — useful for routing decisions
# when summary and specific routes overlap.
output "matches" {
  value = provider::burnham::cidrs_containing_ip(
    "10.0.1.5",
    ["10.0.0.0/8", "10.0.1.0/24", "192.168.0.0/16"],
  )
  # → ["10.0.0.0/8", "10.0.1.0/24"]
}
