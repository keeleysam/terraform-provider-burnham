# Do two CIDRs share any address?
output "conflict" {
  value = provider::burnham::cidr_overlaps("10.0.0.0/24", "10.0.0.128/25")
}
