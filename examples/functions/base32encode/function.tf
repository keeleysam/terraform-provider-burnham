// base32encode: RFC 4648 base32 (core has no base32). Default: standard, padded.
// Options select the extended-hex alphabet and/or unpadded output (e.g. TOTP secrets).
output "secret_b32" {
  value = provider::burnham::base32encode(provider::burnham::hexdecode(var.totp_seed_hex), { padding = false })
}

variable "totp_seed_hex" {
  type    = string
  default = "3132333435363738393031323334353637383930"
}
