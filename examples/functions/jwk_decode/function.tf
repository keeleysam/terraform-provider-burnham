// Convert a JWK object back to PEM. Round-trips with jwk_encode: encoding a key and decoding it returns an equivalent key.
locals {
  ec_key = provider::burnham::ecdsa_p256_key_from_seed("jwk-decode-seed")
  jwk    = provider::burnham::jwk_encode(local.ec_key)
}

output "pem" {
  value = provider::burnham::jwk_decode(local.jwk)
  // → "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n" (equivalent to local.ec_key)
}

// A public JWK decodes to a PUBLIC KEY block.
output "public_pem" {
  value = provider::burnham::jwk_decode(
    provider::burnham::jwk_encode(provider::burnham::x509_self_sign(
      local.ec_key, "example", "0102030405060708", "2001-01-01T00:00:00Z", "2099-01-01T00:00:00Z",
    ))
  )
  // → "-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----\n"
}
