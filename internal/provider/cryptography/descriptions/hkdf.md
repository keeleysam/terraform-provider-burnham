Derives `length` bytes of deterministic key material from `secret` using the [RFC 5869](https://www.rfc-editor.org/rfc/rfc5869) Extract-then-Expand HKDF construction, and returns them hex-encoded. Use it to stretch a single master secret into many stable subkeys (for example one per tenant) without storing each derived value.

The two RFC 5869 steps:

- `Extract`: PRK = HMAC-`algorithm`(`salt`, `secret`)
- `Expand`: take `length` bytes from the output stream keyed by PRK and seeded with `info`

%s

HKDF underpins TLS 1.3, the Signal protocol, and roughly every modern key-derivation pipeline. Backed by [`golang.org/x/crypto/hkdf`](https://pkg.go.dev/golang.org/x/crypto/hkdf).