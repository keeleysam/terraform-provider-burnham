/*
Decode an RFC 8771 I-DUNNO string back to its IPv4 or IPv6 address.

Walks the input codepoint-by-codepoint, infers each codepoint's UTF-8 byte length from its numeric value (per RFC §3 Table 1), takes that many of the codepoint's low-order bits, concatenates, and trims the trailing padding. The total bit-payload disambiguates IPv4 (32–52 bits) from IPv6 (128–148 bits); those ranges don't overlap so no hint is required.

Returns the address in canonical form: dotted-quad for IPv4, RFC 5952 lowercase colon-hex for IPv6.

RFC §3.2 of the original spec says deforming is "intentionally omitted" because "the machines will know how to do it, and by definition humans SHOULD NOT attempt the process." This is the machines knowing how to do it.
*/

output "rfc_8771_example_decoded" {
  // The four-codepoint string from RFC 8771 §5 → "198.51.100.164".
  value = provider::burnham::ip_idunno_decode("clҤ")
}

// Round-trip an IPv6 address through encode + decode.
output "round_trip" {
  value = provider::burnham::ip_idunno_decode(provider::burnham::ip_idunno_encode("2001:db8::1"))
  // → "2001:db8::1"
}
