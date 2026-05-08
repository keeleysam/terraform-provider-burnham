// Decode one or more PEM blocks into a list of {type, headers, base64_body} objects. Body stays base64 so it round-trips bit-exact through `base64decode`.
locals {
  bundle = <<-EOT
    -----BEGIN CERTIFICATE-----
    MIIB...
    -----END CERTIFICATE-----
    -----BEGIN PRIVATE KEY-----
    MII...
    -----END PRIVATE KEY-----
  EOT
}

output "block_types" {
  value = [for b in provider::burnham::pem_decode(local.bundle) : b.type]
  // → ["CERTIFICATE", "PRIVATE KEY"]   (assuming both blocks are well-formed)
}
