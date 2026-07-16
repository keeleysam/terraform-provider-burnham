Computes the keyed-hash message authentication code ([HMAC](https://www.rfc-editor.org/rfc/rfc2104), RFC 2104) of `message` under `key`, and returns it hex-encoded. Reach for it at boundaries that expect a signed payload: webhook signatures, stable per-tenant tokens, or CSRF cookie validation.

`algorithm` selects the underlying hash:

- `"sha1"`: RFC 2104 / FIPS 180-4 (legacy; do not pick for new designs)
- `"sha224"`, `"sha256"`, `"sha384"`, `"sha512"`: FIPS 180-4 SHA-2 family
- `"sha512_224"`, `"sha512_256"`: truncated SHA-512 variants

%s

~> **Note:** This is a derivation, not a MAC verifier. To validate a MAC, compute the expected value and `==`-compare it in HCL.