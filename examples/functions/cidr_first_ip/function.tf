# Network address (all host bits zero). Normalizes input.
output "subnet_base" {
  value = provider::burnham::cidr_first_ip("10.0.0.7/24")
  # → "10.0.0.0"
}
