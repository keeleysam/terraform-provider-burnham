Mints a compact [JWS](https://www.rfc-editor.org/rfc/rfc7515) / [JWT](https://www.rfc-editor.org/rfc/rfc7519), returning the `header.payload.signature` string. Reach for it to sign service-account assertions, short-lived API tokens, or webhook payloads directly in a plan.

`claims` is the JWT payload, an arbitrary object. The registered claim names (`iss`, `sub`, `aud`, `exp`, `iat`, `nbf`, `jti`) are treated as ordinary members: this function never reads the wall clock, so if you want `exp` / `iat` / `nbf` you must supply them yourself (compute them upstream from `plantimestamp()` or a variable).

`algorithm` selects the signature and the key type:

- `HS256`, `HS384`, `HS512`: HMAC. `key` is the shared secret, as raw bytes.
- `ES256`: ECDSA on P-256 with SHA-256. `key` is a PEM EC private key. Signing is [RFC 6979](https://www.rfc-editor.org/rfc/rfc6979) deterministic, and the JWS signature is the fixed 64-byte `R||S` form per [RFC 7518](https://www.rfc-editor.org/rfc/rfc7518).
- `EdDSA`: Ed25519. `key` is a PEM Ed25519 private key. Deterministic by [RFC 8032](https://www.rfc-editor.org/rfc/rfc8032).
- `RS256`, `RS384`, `RS512`: RSASSA-PKCS1-v1_5. `key` is a PEM RSA private key. Deterministic.

`options` (optional) is an object of extra header fields merged into the JWS header, for example `{ kid = "2025-06", typ = "JWT" }`. The `alg` header is always set from `algorithm`.

-> Every algorithm here is deterministic: the same `claims`, `algorithm`, `key`, and `options` always produce the same token. RSASSA-PSS is intentionally not offered because its signatures are randomised.

~> **Not encryption.** A JWT payload is signed, not hidden. Anyone holding the token can read the claims with `jwt_decode`. Do not put secrets in `claims`.