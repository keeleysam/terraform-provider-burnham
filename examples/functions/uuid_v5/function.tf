// Deterministic name-based UUIDs (RFC 9562 §5.5). Same (namespace, name) always returns the same UUID. Useful for stable IDs derived from human-readable names.
output "from_dns_name" {
  value = provider::burnham::uuid_v5("dns", "example.com")
  // → "cfbff0d1-9375-5685-968c-48ce8b15ae17"
}

output "from_url" {
  value = provider::burnham::uuid_v5("url", "https://example.com")
  // → "4fd35a71-71ef-5a55-a9d9-aa75c889a6d0"
}

// You can also pass a literal namespace UUID. Same RFC 4122 namespace as "dns" → same output.
output "from_literal_namespace" {
  value = provider::burnham::uuid_v5("6ba7b810-9dad-11d1-80b4-00c04fd430c8", "example.com")
  // → "cfbff0d1-9375-5685-968c-48ce8b15ae17"
}
