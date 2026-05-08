/*
Deterministic Nano ID — short, URL-safe identifiers derived from a seed. Same seed → same ID. Default is 21 characters from the URL-safe alphabet `_-0-9A-Za-z`, matching upstream nanoid.

Use a per-resource seed when you want unique-per-resource IDs that stay stable across plans.
*/
output "default" {
  value = provider::burnham::nanoid("env-prod")
  // → 21-char string from the URL-safe alphabet, e.g. "3WiSbLYRP4_xQAYVk2DcN"
}

output "shorter" {
  value = provider::burnham::nanoid("env-prod", { size = 8 })
}

output "digits_only" {
  value = provider::burnham::nanoid("env-prod", { alphabet = "0123456789", size = 6 })
}
