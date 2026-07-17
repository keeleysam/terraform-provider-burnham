Converts a [JWK](https://www.rfc-editor.org/rfc/rfc7517) object back to a PEM key. The round-trip pair for [`jwk_encode`](#function-jwk_encode).

A public JWK becomes a PKIX `PUBLIC KEY`; a private JWK becomes a PKCS#8 `PRIVATE KEY`. EC, RSA, and Ed25519 (OKP) keys are supported.

`options` is reserved for future use; no options are currently defined.

-> Feeding `jwk_encode` output straight back into `jwk_decode` returns a key equivalent to the one you started with.