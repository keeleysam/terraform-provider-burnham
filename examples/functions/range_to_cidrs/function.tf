// Convert an inclusive IP range to the minimal CIDR list. Cloud feeds often publish ranges this way.
output "range" {
  value = provider::burnham::range_to_cidrs("10.0.0.1", "10.0.0.6")
  // → ["10.0.0.1/32", "10.0.0.2/31", "10.0.0.4/31", "10.0.0.6/32"]
}
