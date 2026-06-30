// etob — RFC 1751 "english to bytes": recover the key from its words.
// Matching is case-insensitive and the embedded parity bits are verified.
output "key_hex" {
  value = provider::burnham::etob("TIDE ITCH SLOW REIN RULE MOT")
  // → "eb33f77ee73d4053"
}
