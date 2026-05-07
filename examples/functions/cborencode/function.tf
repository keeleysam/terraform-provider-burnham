// Encode a value as CBOR (RFC 8949) — returns a base64 string. Output uses Core Deterministic Encoding so the same input produces byte-identical output.
output "blob" {
  value = provider::burnham::cborencode({ name = "alice", count = 3 })
  // → "omRuYW1lZWFsaWNlZWNvdW50Aw=="
}
