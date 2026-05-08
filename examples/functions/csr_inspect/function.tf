// Decode a PKCS #10 CSR into a structured object. Use it to assert at plan time that a CSR's CN and SANs match what the issuing CA expects.
variable "csr_pem" { type = string }

output "subject" {
  value = provider::burnham::csr_inspect(var.csr_pem).subject
}

output "requested_sans" {
  value = provider::burnham::csr_inspect(var.csr_pem).dns_names
}
