<!-- Edit here: this is the MarkdownDescription source for the burnham jwk_encode function. docs/functions/jwk_encode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Converts a PEM key into a [JWK](https://www.rfc-editor.org/rfc/rfc7517) object. Accepts a public key (`PUBLIC KEY` or `CERTIFICATE`) or a private key (`PRIVATE KEY`, `EC PRIVATE KEY`, `RSA PRIVATE KEY`), across EC (P-256/384/521), RSA, and Ed25519.

The returned object is the JWK with the standard members for the key type: `kty` plus `crv`/`x`/`y` for EC, `n`/`e` for RSA, `crv`/`x` for Ed25519 (OKP), and the private members when a private key is supplied.

`options` (optional) sets JWK metadata:

- `kid`: key ID.
- `use`: intended use, for example `sig` or `enc`.
- `alg`: the algorithm the key is for, for example `ES256`.

Round-trips with [`jwk_decode`](#function-jwk_decode). To publish a public key set, feed several JWKs to [`jwks`](#function-jwks).

~> Encoding a **private** key emits a JWK containing the private key material. Only publish public JWKs (`jwk_encode` on a `PUBLIC KEY` or `CERTIFICATE`).