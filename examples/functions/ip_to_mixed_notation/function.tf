// Format an IPv6 address with the last 32 bits in dotted-decimal — makes embedded IPv4 visible at a glance.
output "human_readable" {
  value = provider::burnham::ip_to_mixed_notation("64:ff9b::c000:201")
  // → "64:ff9b::192.0.2.1"
}
