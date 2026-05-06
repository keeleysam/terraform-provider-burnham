// Carve reserved ranges out of a parent allocation. Result is auto-merged.
output "available" {
  value = provider::burnham::cidr_subtract(["10.0.0.0/22"], ["10.0.1.0/24"])
  // → ["10.0.0.0/24", "10.0.2.0/23"]
}
