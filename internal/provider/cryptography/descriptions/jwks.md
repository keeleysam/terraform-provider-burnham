<!-- Edit here: this is the MarkdownDescription source for the burnham jwks function. docs/functions/jwks.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Assembles a [JWK Set](https://www.rfc-editor.org/rfc/rfc7517#section-5) (JWKS): given a list of keys, returns `{ keys = [ ...jwk... ] }`, the shape a JWKS endpoint serves.

Each element of `keys` is either a PEM string or a JWK object. PEM inputs are converted to JWKs; JWK objects pass through normalised. Mix the two freely.

Typical use is publishing the public keys a service signs tokens with, so relying parties can fetch and verify:

```hcl
jsonencode(provider::burnham::jwks([
  provider::burnham::jwk_encode(file("signer1.pub.pem"), { kid = "2025-06" }),
  provider::burnham::jwk_encode(file("signer2.pub.pem"), { kid = "2025-07" }),
]))
```

~> Only put **public** keys in a published JWKS. If you pass a private key or a private JWK, its private members land in the set.