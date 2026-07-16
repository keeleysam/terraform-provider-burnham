Parses any UUID and returns a fixed-shape object describing it.

The returned object has these attributes:

- `version`: integer in `[0, 15]`. Typically 1, 3, 4, 5, 6, 7, or 8.
- `variant`: one of `"RFC 4122"` (covers RFC 9562), `"NCS"`, `"Microsoft"`, `"Future"`, `"Invalid"`.
- `timestamp`: RFC 3339 timestamp encoded in the UUID for v1, v6, and v7. `null` for other versions, where no timestamp is encoded.
- `unix_ts_ms`: the raw 48-bit Unix-millisecond field for v7 UUIDs. `null` for other versions.

~> **Note:** Fails the plan when the input is not a valid UUID string.