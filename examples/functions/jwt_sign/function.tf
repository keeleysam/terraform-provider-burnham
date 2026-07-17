/*
Mint a compact JWS / JWT. The signature is deterministic: the same claims, algorithm, key, and options always produce the same token, so a token in Terraform state does not churn between plans.

`exp` is caller-supplied. This function never reads the clock, so derive time claims upstream (here a fixed value; in practice compute from a variable or plantimestamp()).
*/

// HS256 with a shared secret, plus a `typ` header field via options.
output "hs256" {
  value = provider::burnham::jwt_sign(
    { sub = "alice", role = "admin", exp = 1735689600 },
    "HS256",
    "topsecret",
    { typ = "JWT" },
  )
  // -> "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MzU2ODk2MDAsInJvbGUiOiJhZG1pbiIsInN1YiI6ImFsaWNlIn0.iSfdZFITiXsyvzRTshLDuWiIaRDsw1XWuk73vxFCwp0"
}

// ES256 with a deterministic key derived from a seed: identity and signature are both a pure function of the inputs.
locals {
  ec_key = provider::burnham::ecdsa_p256_key_from_seed("signing-identity-seed")
}

output "es256" {
  value = provider::burnham::jwt_sign(
    { sub = "service-account", aud = "https://api.example.com" },
    "ES256",
    local.ec_key,
    { kid = provider::burnham::jwk_thumbprint(local.ec_key) },
  )
}
