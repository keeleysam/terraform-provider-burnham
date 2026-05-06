// Embed an IPv4 into a NAT64 prefix. Mixed notation by default.
output "nat64" {
  value = provider::burnham::nat64_synthesize("192.0.2.1", "64:ff9b::/96")
  // → "64:ff9b::192.0.2.1"
}
