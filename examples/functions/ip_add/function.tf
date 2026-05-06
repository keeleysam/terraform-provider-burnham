// Conventional addresses derived from a subnet base.
locals {
  subnet_base = provider::burnham::cidr_first_ip("10.0.1.0/24") // "10.0.1.0"
  gateway     = provider::burnham::ip_add(local.subnet_base, 1) // "10.0.1.1"
  dns         = provider::burnham::ip_add(local.subnet_base, 2) // "10.0.1.2"
}
