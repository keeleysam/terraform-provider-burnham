<!-- Edit here: this is the MarkdownDescription source for the burnham uuid_v5 function. docs/functions/uuid_v5.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns a [version 5 UUID](https://www.rfc-editor.org/rfc/rfc9562#name-uuid-version-5): the SHA-1 hash of a namespace UUID concatenated with a name.

This function is **deterministic**: the same `(namespace, name)` pair always returns the same UUID, with no randomness involved. That makes it ideal for stable, plan-time IDs derived from human-meaningful names.

`namespace` may be either a well-formed UUID string or one of the four predefined RFC 4122 short names, which map to the namespace UUIDs from [RFC 4122 Appendix C](https://www.rfc-editor.org/rfc/rfc4122#appendix-C):

- `"dns"`
- `"url"`
- `"oid"`
- `"x500"`

-> **Note:** RFC 9562 keeps v5 as the standard name-based UUID (recommending it over v3, since v5 uses SHA-1 rather than MD5) and points to v8 only if you need a different hash algorithm. v5 remains the broadly supported deterministic-UUID option and is what most existing systems consume.