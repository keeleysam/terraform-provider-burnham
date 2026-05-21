/*
Build a deterministic self-signed X.509 certificate from a PEM-encoded ECDSA P-256 private key. Signs with RFC 6979 deterministic ECDSA so the cert's bytes are stable across runs. Output is PEM.

Compliance posture (RFC 5280): v3 cert, positive serial ≤ 20 octets (§4.1.2.2), UTCTime/GeneralizedTime split at year 2050 (§4.1.2.5), BasicConstraints critical with cA=FALSE (§4.2.1.9).

Paired with `ecdsa_p256_key_from_seed` the full chain seed → key → cert is deterministic with no random state at any step.
*/
locals {
  seed = sha512(file("payload.bin"))
  key  = provider::burnham::ecdsa_p256_key_from_seed(local.seed)
  // Derive an independent serial from the same seed via HKDF for domain separation. We ask for 10 bytes (= 20 hex chars; `hkdf` returns hex, and `x509_self_sign` treats the parameter as raw bytes per the burnham byte-handling convention) so the resulting serial lands at exactly the RFC 5280 §4.1.2.2 20-octet cap. Asking for more would overflow.
  serial = provider::burnham::hkdf("sha256", local.seed, "", "signer-serial", 10)
}

output "signer_cert" {
  value = provider::burnham::x509_self_sign(
    local.key,
    "signer.example",
    local.serial,
    "2001-01-01T00:00:00Z",
    "2099-01-01T00:00:00Z",
  )
  // → "-----BEGIN CERTIFICATE-----\nMIIByzCCAXGgAwIBAgIUMASm…\n-----END CERTIFICATE-----\n"
}
