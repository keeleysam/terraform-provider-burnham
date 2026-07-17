/*
Verify a JWT signature. With no `now` option the check is signature-only and stays pure. Supply `now` (unix seconds or an RFC 3339 string) to additionally enforce `exp` / `nbf`.
*/

locals {
  token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MzU2ODk2MDAsInJvbGUiOiJhZG1pbiIsInN1YiI6ImFsaWNlIn0.iSfdZFITiXsyvzRTshLDuWiIaRDsw1XWuk73vxFCwp0"
}

// Signature only.
output "signature_valid" {
  value = provider::burnham::jwt_verify(local.token, "topsecret").valid
  // -> true
}

// A different secret fails the signature check (result is false, not an error).
output "wrong_secret" {
  value = provider::burnham::jwt_verify(local.token, "not-the-secret").valid
  // -> false
}

// With `now` past `exp` (1735689600), the token is expired.
output "expired" {
  value = provider::burnham::jwt_verify(local.token, "topsecret", { now = 1900000000 }).valid
  // -> false
}
