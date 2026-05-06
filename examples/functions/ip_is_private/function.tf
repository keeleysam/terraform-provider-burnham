# RFC 1918 + RFC 6598 (CGNAT) + RFC 4193 (ULA) + loopback + link-local.
output "internal" {
  value = provider::burnham::ip_is_private("100.64.0.1")  # CGNAT
  # → true
}
