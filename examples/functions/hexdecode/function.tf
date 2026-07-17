// hexdecode: hex back to bytes. Lenient: case-insensitive, ASCII whitespace ignored.
// Closes the gap that core leaves: now hmac/hkdf can take a hex key directly.
output "mac" {
  value = provider::burnham::hmac("sha256", provider::burnham::hexdecode(var.signing_key_hex), "payload")
}

variable "signing_key_hex" {
  type    = string
  default = "0f1e2d3c4b5a69788796a5b4c3d2e1f0"
}
