// Hex fingerprint of a certificate's DER bytes. Same value as `openssl x509 -fingerprint -sha256` minus the colons.
variable "tls_cert_pem" { type = string }

output "sha256_pin" {
  value = provider::burnham::x509_fingerprint(var.tls_cert_pem, "sha256")
  // → e.g. "6d2d325a319a26c8d89f417fc543d2673bdce5f9ba2e4ae2bdc6f409f0e346cc"
}
