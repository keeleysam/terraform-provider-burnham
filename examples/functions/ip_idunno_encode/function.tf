/*
Encode an IP address as RFC 8771 I-DUNNO, the Internationalized Deliberately Unreadable Network Notation.

The encoding packs the address bits (32 for IPv4, 128 for IPv6) into a sequence of Unicode codepoints whose UTF-8 byte lengths carry the bits per RFC §3 Table 1. The result satisfies the §4.1 Minimum Confusion Level: at least one multi-octet UTF-8 sequence and at least one IDNA2008-DISALLOWED character.

The function is deterministic for a given input. RFC §5's worked example (`198.51.100.164` → U+0063, U+000C, U+006C, U+04A4) round-trips through this encoder exactly.

Dual-stack: works on both IPv4 and IPv6. The bit-packing operates on the raw network-byte-order bitstring, so address family is just a length difference.

Pair with `ip_idunno_decode` to recover the original address. RFC §3.2 says deforming "is intentionally omitted" because "humans SHOULD NOT attempt the process", but the machines DO know how to do it.
*/

output "rfc_8771_example" {
  // The exact output from RFC 8771 §5: U+0063 (c), U+000C (FF), U+006C (l), U+04A4 (Ҥ).
  value = provider::burnham::ip_idunno_encode("198.51.100.164")
  // → "clҤ"
}

output "ipv6" {
  // Some 8-codepoint UTF-8 string. The exact output depends on which layout the encoder picks for this address.
  value = provider::burnham::ip_idunno_encode("2001:db8::1")
}

output "loopback" {
  value = provider::burnham::ip_idunno_encode("127.0.0.1")
}
