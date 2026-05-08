/*
HKDF (RFC 5869) — derive multiple subkeys deterministically from a single master secret. Returns hex-encoded bytes; if a downstream consumer needs raw bytes, decode the hex outside Terraform (Burnham does not currently ship a `hex_decode` helper).
*/
output "tenant_a" {
  value = provider::burnham::hkdf("sha256", "master-secret", "deployment-salt", "tenant=a", 32)
}

output "tenant_b" {
  value = provider::burnham::hkdf("sha256", "master-secret", "deployment-salt", "tenant=b", 32)
  // Different `info` → different bytes; same secret + salt would re-derive identically each plan.
}
