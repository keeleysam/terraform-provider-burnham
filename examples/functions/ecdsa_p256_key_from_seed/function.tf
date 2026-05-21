/*
ECDSA P-256 key from a deterministic seed (PEM PKCS#8 output). Same seed → same key, every run. Use it when you want a stable signing identity derived from a checked-in secret or an input artefact instead of randomly generated and stored in state.

Pair with `x509_self_sign` and `pkcs7_sign` to build deterministic signing pipelines that are byte-stable across Terraform plans.
*/
output "signer_key" {
  value     = provider::burnham::ecdsa_p256_key_from_seed(sha512(file("payload.bin")))
  sensitive = true
  // → "-----BEGIN PRIVATE KEY-----\nMIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEH…\n-----END PRIVATE KEY-----\n"
}

// Different seeds produce independent keys; same seed always re-derives identically.
output "tenant_a_key" {
  value     = provider::burnham::ecdsa_p256_key_from_seed("tenant=a|deployment=prod")
  sensitive = true
}
