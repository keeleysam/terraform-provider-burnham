// Encode a value as MessagePack — returns a base64 string (HCL strings are UTF-8 only, so binary outputs are base64-wrapped).
output "blob" {
  value = provider::burnham::msgpackencode({ name = "alice", count = 3 })
  // → "gqVjb3VudAOkbmFtZaVhbGljZQ=="
}
