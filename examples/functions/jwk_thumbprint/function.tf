// The RFC 7638 canonical thumbprint of a key, base64url-encoded. This is the standard, stable value for a `kid`. Accepts a PEM string or a JWK object.
locals {
  ec_key = provider::burnham::ecdsa_p256_key_from_seed("kid-seed")
}

// From a PEM key, default SHA-256.
output "kid" {
  value = provider::burnham::jwk_thumbprint(local.ec_key)
  // -> a base64url string, stable for this key, for example "kZTk...c9Xs"
}

// From a JWK object, SHA-1 digest.
output "sha1_thumbprint" {
  value = provider::burnham::jwk_thumbprint(provider::burnham::jwk_encode(local.ec_key), "SHA-1")
}
