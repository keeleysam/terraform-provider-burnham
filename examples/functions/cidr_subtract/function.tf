# Carve reserved ranges out of a parent allocation.
output "available" {
  value = provider::burnham::cidr_subtract(["10.0.0.0/8"], ["10.1.0.0/16"])
}
