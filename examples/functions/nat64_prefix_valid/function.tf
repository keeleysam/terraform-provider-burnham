// Validate a NAT64 prefix per RFC 6052: IPv6, length in {32, 40, 48, 56, 64, 96}, u-octet zero.
output "ok" {
  value = provider::burnham::nat64_prefix_valid("64:ff9b::/96")
  // → true
}
