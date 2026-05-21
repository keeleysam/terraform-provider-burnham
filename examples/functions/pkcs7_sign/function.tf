/*
CMS / PKCS#7 sign bytes with an ECDSA P-256 identity (deterministic, no signed attrs). Output is base64-encoded DER.

Produces an RFC 5652 SignedData ContentInfo: `id-data` encapsulated content, no signed attributes (§5.3 permits omitting them when the content type is `id-data`), embedded signer cert, signature via RFC 6979 deterministic ECDSA-with-SHA256.

Apple's configuration-profile installer accepts this exact shape on macOS, and Jamf passes signed profiles through unchanged.
*/

// "Real identity" mode: caller-supplied key + cert (e.g. a CA-issued signer). Determinism still applies — same (data, key, cert) → same bytes.
output "signed_with_real_identity" {
  value = provider::burnham::pkcs7_sign(
    file("payload.bin"),
    file("signer.key.pem"),
    file("signer.cert.pem"),
  )
}

// "Derive everything from the input" mode: identity is a function of the payload, so two callers with the same payload always produce the same signed bytes — useful for Terraform-driven workflows that need plan-to-apply stability without managing long-lived signing keys.
locals {
  seed = sha512(file("payload.bin"))
  key  = provider::burnham::ecdsa_p256_key_from_seed(local.seed)
  cert = provider::burnham::x509_self_sign(
    local.key,
    "signer.example",
    // 10 bytes from HKDF → 20 hex chars; `x509_self_sign` treats those as raw bytes → 20-octet serial, exactly at the RFC 5280 §4.1.2.2 cap.
    provider::burnham::hkdf("sha256", local.seed, "", "signer-serial", 10),
    "2001-01-01T00:00:00Z",
    "2099-01-01T00:00:00Z",
  )
}

output "signed_with_derived_identity" {
  value = provider::burnham::pkcs7_sign(file("payload.bin"), local.key, local.cert)
}
