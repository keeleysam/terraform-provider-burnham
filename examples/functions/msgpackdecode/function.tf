// Decode a base64-encoded MessagePack blob.
output "decoded" {
  value = provider::burnham::msgpackdecode("gqVjb3VudAOkbmFtZaVhbGljZQ==")
  // → { name = "alice", count = 3 }
}
