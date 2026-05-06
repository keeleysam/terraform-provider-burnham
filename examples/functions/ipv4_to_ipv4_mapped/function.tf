// RFC 4291 §2.5.5.2 IPv4-mapped IPv6 form, in mixed notation.
output "mapped" {
  value = provider::burnham::ipv4_to_ipv4_mapped("192.0.2.1")
  // → "::ffff:192.0.2.1"
}
