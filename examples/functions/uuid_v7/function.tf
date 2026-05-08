/*
Deterministic time-ordered UUIDs (RFC 9562 §5.7). The 48-bit Unix-millisecond timestamp lives in the leading bytes, making v7 UUIDs lexicographically sortable by creation time — much better than v4 for database keys, log IDs, and ordered storage.

This implementation derives the 74 random-ish bits via HMAC-SHA-256 keyed by `entropy`, so a stable (timestamp, entropy) pair always returns the same UUID. Use a per-resource `entropy` to get unique-per-resource sortable IDs that don't churn the plan.
*/
output "stable_id" {
  value = provider::burnham::uuid_v7("2026-05-08T12:00:00Z", "stable-key")
}

// Two outputs from the same timestamp but different entropy diverge in the random portion while keeping the same time-ordered prefix.
output "different_entropy" {
  value = provider::burnham::uuid_v7("2026-05-08T12:00:00Z", "another-key")
}
