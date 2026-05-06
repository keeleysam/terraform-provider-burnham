# Plain integer prefix length, for APIs that don't take CIDR notation.
output "bits" {
  value = provider::burnham::cidr_prefix_length("10.0.0.0/23")
  # → 23
}
