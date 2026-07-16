// hexdecode: hex back to bytes. Lenient: case-insensitive, ASCII whitespace ignored.
// Closes the gap that core leaves: now hmac/hkdf can take a hex key directly.
output "mac" {
  value = provider::burnham::hmac("sha256", provider::burnham::hexdecode(var.signing_key_hex), "payload")
}

variable "signing_key_hex" { type = string }
