// Decode a JWT into its header and payload WITHOUT verifying the signature. Handy for reading a `kid` or `iss` before selecting a verification key. Never trust these claims until jwt_verify confirms the signature.
locals {
  token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MzU2ODk2MDAsInJvbGUiOiJhZG1pbiIsInN1YiI6ImFsaWNlIn0.iSfdZFITiXsyvzRTshLDuWiIaRDsw1XWuk73vxFCwp0"
}

output "header" {
  value = provider::burnham::jwt_decode(local.token).header
  // -> { alg = "HS256", typ = "JWT" }
}

output "subject" {
  value = provider::burnham::jwt_decode(local.token).payload.sub
  // -> "alice"
}
