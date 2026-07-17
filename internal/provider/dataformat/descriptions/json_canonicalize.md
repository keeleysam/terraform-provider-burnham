Serializes a value as canonical JSON per [RFC 8785](https://www.rfc-editor.org/rfc/rfc8785) (the JSON Canonicalization Scheme, JCS). Canonical JSON is the exact byte sequence you feed to a signing or MAC primitive so that the same logical value always hashes to the same digest, regardless of key order or whitespace.

Reach for it when the bytes matter:

- The input to [`hmac`](#function-hmac), [`hkdf`](#function-hkdf), or [`pkcs7_sign`](#function-pkcs7_sign).
- Building a stable content digest or ETag for a structured value.
- Any place two parties must agree on identical bytes for the same JSON document.

JCS produces deterministic output by:

- Sorting object member names lexicographically by their UTF-16 code units.
- Removing all insignificant whitespace.
- Serializing numbers under the I-JSON / ECMAScript `Number` (IEEE 754 double) rules.
- Applying minimal, canonical string escaping.

~> **Numbers are doubles.** RFC 8785 serializes every number as an ECMAScript double (IEEE 754 64-bit). Integers beyond 2^53 lose exactness and follow the spec's double serialization (for example a value near 2^63 renders with trailing zeroes). If you need exact large integers in a signed document, carry them as strings.

-> The result is a string. Canonicalizing the same value twice always yields byte-identical output, which is the entire point.