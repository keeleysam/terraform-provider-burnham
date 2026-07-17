// Convert a PEM key to a JWK object, optionally attaching kid / use / alg metadata. Pass a public key (or certificate) to produce a publishable public JWK.
locals {
  ec_key = provider::burnham::ecdsa_p256_key_from_seed("jwk-example-seed")
}

output "jwk" {
  value = provider::burnham::jwk_encode(local.ec_key, {
    use = "sig"
    alg = "ES256"
    kid = provider::burnham::jwk_thumbprint(local.ec_key)
  })
  /* → an object like:
     {
       kty = "EC"
       crv = "P-256"
       x   = "..."
       y   = "..."
       d   = "..."   # present because a private key was supplied
       use = "sig"
       alg = "ES256"
       kid = "..."
     }
  */
}
