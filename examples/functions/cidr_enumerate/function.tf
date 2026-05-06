// Every subnet of a target size within a parent.
output "az_subnets" {
  value = provider::burnham::cidr_enumerate("10.0.0.0/24", 2)
  // → ["10.0.0.0/26", "10.0.0.64/26", "10.0.0.128/26", "10.0.0.192/26"]
}
