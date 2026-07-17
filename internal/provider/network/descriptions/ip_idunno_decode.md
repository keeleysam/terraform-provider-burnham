<!-- Edit here: this is the MarkdownDescription source for the burnham ip_idunno_decode function. docs/functions/ip_idunno_decode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Reverses [`ip_idunno_encode`](#function-ip_idunno_encode), turning an RFC 8771 I-DUNNO string back into a normal IP address.

It walks the input codepoint-by-codepoint, determines each codepoint's UTF-8 byte length, and accumulates the corresponding number of low-order bits per RFC 8771 §3 Table 1:

- 1-byte = 7 bits
- 2-byte = 11 bits
- 3-byte = 16 bits
- 4-byte = 21 bits

The total bit-payload disambiguates IPv4 (32–52 codepoint bits: 32 address + ≤ 20 padding) from IPv6 (128–148); those ranges don't overlap, so the decoder doesn't need a hint.

Returns the address in canonical text form: dotted-quad for IPv4, [RFC 5952](https://www.rfc-editor.org/rfc/rfc5952.html) lowercase colon-hex for IPv6. Use `provider::burnham::ip_version(...)` if you need to branch on the family afterwards.

-> **Note:** RFC §3.2 says deforming "is intentionally omitted. The machines will know how to do it, and by definition humans SHOULD NOT attempt the process." This is the machines knowing how to do it.

~> **Note:** Fails the plan when the input isn't valid UTF-8, or has a total bit-payload that doesn't match either IPv4 or IPv6.