<!-- Edit here: this is the MarkdownDescription source for the burnham ecdsa_p256_key_from_seed function. docs/functions/ecdsa_p256_key_from_seed.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Derives a `secp256r1` (P-256) private key deterministically from `seed` and returns it as PEM PKCS#8. The same `seed` always produces the same key, so you get a stable signing identity from a checked-in secret or input artefact rather than a randomly generated key stored in state.

How the scalar is derived:

- Stretch `seed` to 48 bytes with HKDF-SHA256 (info string `%q`).
- Reduce that value modulo (n-1) and add 1, landing uniformly in [1, n-1].
- Assemble the resulting scalar into the P-256 private key.

Pair with [`x509_self_sign`](#function-x509_self_sign) and [`pkcs7_sign`](#function-pkcs7_sign) to build deterministic signing pipelines that are byte-stable across Terraform plans.

%s