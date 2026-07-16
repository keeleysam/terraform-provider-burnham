// base64encode: RFC 4648. No options = standard padded (same as core base64encode).
// Options select URL-safe (§5) and/or unpadded output, e.g. for JWT/JOSE values.
output "token" {
  value = provider::burnham::base64encode(var.payload, { url_safe = true, padding = false })
}

variable "payload" { type = string }
