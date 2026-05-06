// Recover the IPv4 from a NAT64 IPv6 address. Default extracts the last 32 bits — correct for any /96 prefix including 64:ff9b::/96.
output "ipv4" {
  value = provider::burnham::nat64_extract("64:ff9b::192.0.2.1")
  // → "192.0.2.1"
}
