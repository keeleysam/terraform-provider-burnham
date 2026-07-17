<!-- Edit here: this is the MarkdownDescription source for the burnham asn1_decode function. docs/functions/asn1_decode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Decodes ASN.1 DER bytes into a recursive object tree you can walk in HCL, for pulling apart an extension payload, a field inside an opaque blob, or any structure Terraform has no native decoder for.

Input is base64-encoded DER bytes, the same shape `pem_decode` returns in `base64_body`, which keeps inputs ASCII-safe inside HCL strings.

Every node in the tree has the same shape:

- `tag`: the BER tag number (`2` for INTEGER, `6` for OBJECT IDENTIFIER, `16` for SEQUENCE, …).
- `class`: `"universal"`, `"application"`, `"context"`, or `"private"`.
- `compound`: `true` for constructed values that hold child nodes; `false` for primitive values.
- `type`: human-readable name for universal-class tags (`"INTEGER"`, `"SEQUENCE"`, `"OBJECT IDENTIFIER"`, …); empty string for non-universal classes.
- `value`: the primitive payload as a string, always `""` when `compound = true`. See the per-tag encoding below.
- `children`: a list of decoded child nodes when `compound = true`; an empty list otherwise (the framework forbids null lists of objects in a recursive-feeling tree).

### Value encoding by tag

The primitive `value` string is encoded according to the node's tag:

- `INTEGER` / `ENUMERATED` → decimal string
- `BOOLEAN` → `"true"` / `"false"`
- `OBJECT IDENTIFIER` → dotted form (`"1.3.6.1.5.5.7.3.1"`)
- `UTF8String` / `PrintableString` / `IA5String` / `NumericString` / `GeneralString` → the string value
- `BMPString` → UTF-8 (decoded from UCS-2 big-endian)
- `T61String` → the string value when all bytes are ASCII; otherwise `"t61_hex:<hex>"` (full ISO 6937 transcoding is intentionally not bundled, so pre-encode as UTF8String upstream if you need legible output)
- `BIT STRING` / `OCTET STRING` → hex
- `UTCTime` / `GeneralizedTime` → RFC 3339 timestamp
- `NULL` → empty string
- other primitives → hex of the raw value bytes

~> **Note:** `value` is always a string, regardless of tag. Even INTEGER and BOOLEAN nodes return their value as text (`"42"`, `"true"`), so convert per-tag with `tonumber(node.value)` or `node.value == "true"`. The single-typed field is what keeps the recursive schema buildable in Terraform: the framework can't express a recursive object type with per-node varying value types.

### Resource limits

The decoder bounds adversarial input:

- The base64 input may be at most 8 MiB (roughly 6 MiB of decoded DER); larger inputs are rejected before base64 decoding.
- Nesting may be at most 64 levels deep. RFC 5280 X.509 nesting fits comfortably under this limit.
- A single decode may produce at most 100,000 nodes. The largest realistic certs sit around 1,000.

~> **Note:** Fails when the bytes are not well-formed DER, when an INTEGER is malformed or not canonically DER-encoded, when a date stamp can't be parsed, or when any of the above limits are exceeded.