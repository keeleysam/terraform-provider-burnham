// Decode a base64-encoded CBOR blob (RFC 8949).
output "decoded" {
  value = provider::burnham::cbordecode("omRuYW1lZWFsaWNlZWNvdW50Aw==")
  // → { name = "alice", count = 3 }
}
