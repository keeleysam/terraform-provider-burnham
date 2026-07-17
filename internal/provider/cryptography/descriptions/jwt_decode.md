<!-- Edit here: this is the MarkdownDescription source for the burnham jwt_decode function. docs/functions/jwt_decode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Decodes a compact [JWS](https://www.rfc-editor.org/rfc/rfc7515) / [JWT](https://www.rfc-editor.org/rfc/rfc7519) into its `header` and `payload` objects, **without verifying the signature**. Use it to inspect a token's claims (for example to read a `kid` or an `iss` before choosing a verification key).

Returns an object:

- `header`: the decoded protected header object (includes `alg`, and whatever else was set).
- `payload`: the decoded claims object.

Round-trips with [`jwt_sign`](#function-jwt_sign): decoding a signed token returns the same header and claims that went in.

~> **This does not verify anything.** The signature segment is ignored. A decoded payload is attacker-controlled until you check it with [`jwt_verify`](#function-jwt_verify). Never make a trust decision on `jwt_decode` output alone.