/*
Decode any RFC 4122 / RFC 9562 UUID into version, variant, and (where encoded) timestamp. Returns a fixed-shape object: `{ version, variant, timestamp, unix_ts_ms }`. Timestamp is null for versions that don't encode one (3, 4, 5, 8); unix_ts_ms is non-null only for v7.
*/
output "v4_random" {
  value = provider::burnham::uuid_inspect("550e8400-e29b-41d4-a716-446655440000")
  // → { version = 4, variant = "RFC 4122", timestamp = null, unix_ts_ms = null }
}

output "v5_name_based" {
  value = provider::burnham::uuid_inspect("cfbff0d1-9375-5685-968c-48ce8b15ae17")
  // → { version = 5, variant = "RFC 4122", timestamp = null, unix_ts_ms = null }
}

// Round-trip a v7 UUID we just generated.
output "v7_round_trip" {
  value = provider::burnham::uuid_inspect(provider::burnham::uuid_v7("2026-05-08T12:00:00Z", "key")).timestamp
  // → "2026-05-08T12:00:00Z"
}
