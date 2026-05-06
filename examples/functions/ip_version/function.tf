// 4 or 6. IPv4-mapped IPv6 (::ffff:1.2.3.4) is treated as IPv4.
output "family" {
  value = provider::burnham::ip_version("2001:db8::1")
  // → 6
}
