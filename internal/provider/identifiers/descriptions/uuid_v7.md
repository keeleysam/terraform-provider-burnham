Returns a [version 7 UUID](https://www.rfc-editor.org/rfc/rfc9562#name-uuid-version-7) with a 48-bit Unix-millisecond timestamp embedded in its leading bytes.

v7 UUIDs are **lexicographically sortable** by creation time, which makes them a much better choice than v4 for database keys, log IDs, and ordered storage.

This function is **deterministic**: the 74 random-ish bits (`rand_a`, `rand_b`) are derived from `entropy` via HMAC-SHA-256, so a stable `(timestamp, entropy)` pair always returns the same UUID. Use this when you want sortable IDs that don't churn the Terraform plan on re-apply. For unique IDs at plan time, give each call a different `entropy` (for example the resource name).

`timestamp` accepts any RFC 3339 / RFC 3339 Nano timestamp, for example `"2026-05-08T12:00:00Z"`. Sub-millisecond precision is truncated; the v7 spec only carries milliseconds.

~> **Note:** Always pass a meaningful `entropy`. The empty string is accepted, but it makes the random bits a fixed function of the timestamp alone: every call sharing that timestamp returns the same UUID, defeating the point of the random fields. Use the resource name, a logical key, or any other per-call string to keep IDs distinct.