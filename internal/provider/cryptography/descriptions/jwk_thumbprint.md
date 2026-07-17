<!-- Edit here: this is the MarkdownDescription source for the burnham jwk_thumbprint function. docs/functions/jwk_thumbprint.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Computes the [RFC 7638](https://www.rfc-editor.org/rfc/rfc7638) canonical JWK thumbprint, base64url-encoded. This is the conventional value for a key's `kid`.

`key` is either a PEM string or a JWK object. The thumbprint is a hash over the JWK's required members (only the members that define the key: `kty` plus the key-type-specific ones), serialised in the lexicographic, whitespace-free canonical form the RFC mandates, so it is stable regardless of member order or extra metadata like `kid` / `use` / `alg`.

`hash` (optional) selects the digest, defaulting to `"SHA-256"`. Also accepts `"SHA-1"`, `"SHA-384"`, and `"SHA-512"`.

-> The same key always yields the same thumbprint, which makes it a natural stable `kid`: set it with `jwk_encode(pem, { kid = jwk_thumbprint(pem) })`.