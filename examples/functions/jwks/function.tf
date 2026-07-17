/*
Assemble a JWK Set (JWKS), the shape a JWKS endpoint serves. Feed a list of PEM strings or JWK objects; mix the two freely. Publish only public keys.
*/

locals {
  key1 = provider::burnham::ecdsa_p256_key_from_seed("signer-2025-06")
  key2 = provider::burnham::ecdsa_p256_key_from_seed("signer-2025-07")
}

output "jwks_json" {
  value = jsonencode(provider::burnham::jwks([
    provider::burnham::jwk_encode(local.key1, { kid = "2025-06", use = "sig" }),
    provider::burnham::jwk_encode(local.key2, { kid = "2025-07", use = "sig" }),
  ]))
  // -> {"keys":[{"crv":"P-256","kid":"2025-06","kty":"EC","use":"sig","x":"...","y":"..."}, ...]}
}
