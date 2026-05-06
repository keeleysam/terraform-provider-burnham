// Tagged-object representation of an Apple plist <data> (binary) element.
output "cert_blob" {
  value = provider::burnham::plistdata("SGVsbG8=")
  // → { __plist_type = "data", value = "SGVsbG8=" }
}
