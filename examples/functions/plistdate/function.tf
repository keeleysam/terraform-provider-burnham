// Tagged-object representation of an Apple plist <date> element.
output "expires" {
  value = provider::burnham::plistdate("2026-06-01T00:00:00Z")
  // → { __plist_type = "date", value = "2026-06-01T00:00:00Z" }
}
