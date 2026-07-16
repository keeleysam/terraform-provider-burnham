/*
Ed25519 key from a deterministic seed (PEM PKCS#8 output). Same seed → same key, every run. Use it when you want a stable signing identity derived from a checked-in secret or an input artefact instead of randomly generated and stored in state.

Ed25519 is naturally deterministic by spec (RFC 8032 §5.1.6), so unlike `ecdsa_p256_key_from_seed` no RFC 6979 wrapper is involved at the signing layer either. Pair with `x509_self_sign` and `pkcs7_sign`, both of which accept Ed25519 keys and dispatch the right algorithm on key type.

Note: macOS configuration-profile signing rejects Ed25519 at the keychain-import layer as of macOS 26.5. Use `ecdsa_p256_key_from_seed` for `.mobileconfig` workflows; Ed25519 is the right pick for everything else (OpenSSL `cms`, container signing, internal tooling).
*/
output "signer_key" {
  value     = provider::burnham::ed25519_key_from_seed(sha512(file("payload.bin")))
  sensitive = true
  // → "-----BEGIN PRIVATE KEY-----\nMC4CAQAwBQYDK2VwBCIEI…\n-----END PRIVATE KEY-----\n"
}

// Different seeds produce independent keys; same seed always re-derives identically.
output "tenant_a_key" {
  value     = provider::burnham::ed25519_key_from_seed("tenant=a|deployment=prod")
  sensitive = true
}
