// HMAC (RFC 2104) — keyed message authentication code, hex-encoded. Useful at the seam where Terraform-rendered values feed a service that expects a signed request body.
output "webhook_signature" {
  value = provider::burnham::hmac("sha256", "shared-secret", "payload-bytes")
}

// SHA-512 for higher-security paths.
output "long_signature" {
  value = provider::burnham::hmac("sha512", "k", "m")
}
