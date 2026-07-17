<!-- Edit here: this is the MarkdownDescription source for the burnham ip_idunno_encode function. docs/functions/ip_idunno_encode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Encodes an IPv4 or IPv6 address into the Internationalized Deliberately Unreadable Network Notation per [RFC 8771](https://www.rfc-editor.org/rfc/rfc8771.html) (April 1, 2020).

The output is a UTF-8 string of Unicode codepoints whose UTF-8 byte lengths carry the address bits per RFC §3 Table 1:

- 1-byte sequence = 7 bits
- 2-byte sequence = 11 bits
- 3-byte sequence = 16 bits
- 4-byte sequence = 21 bits

The encoder is deterministic for a given input and reaches at least the **Minimum Confusion Level** of §4.1 (≥ 1 multi-octet UTF-8 sequence AND ≥ 1 IDNA2008-DISALLOWED character). RFC §5's worked example (`198.51.100.164` → `c\u000Cl\u04A4`, i.e. U+0063, U+000C, U+006C, U+04A4) round-trips through this encoder exactly.

Dual-stack: §3.1 specifies the bitstring length as "32 bits for IPv4; 128 bits for IPv6" and the rest of the spec operates on raw bits, so the same encoder handles both families.

Pair with [`ip_idunno_decode`](#function-ip_idunno_decode) to reverse the transformation.

-> **Note:** RFC §3.2 says deforming "is intentionally omitted" because "humans SHOULD NOT attempt the process"; the decoder is intended for the machines.