# Number of address positions between two IPs (signed).
output "diff" {
  value = provider::burnham::ip_subtract("10.0.0.10", "10.0.0.1")
  # → 9
}
