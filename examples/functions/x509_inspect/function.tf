// Decode an X.509 certificate's metadata into a structured object. Useful for plan-time assertions about expiry, SANs, and key/extended-key usage.
variable "tls_cert_pem" { type = string }

output "expiry" {
  value = provider::burnham::x509_inspect(var.tls_cert_pem).not_after
}

output "sans" {
  value = provider::burnham::x509_inspect(var.tls_cert_pem).dns_names
}

output "is_server_auth" {
  value = contains(provider::burnham::x509_inspect(var.tls_cert_pem).ext_key_usage, "serverAuth")
}
