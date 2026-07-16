Encodes a Terraform object as a [Valve Data Format (VDF)](https://developer.valvesoftware.com/wiki/KeyValues) string. VDF is a nested key-value format built from strings and nested objects.

Value handling:

- Strings and nested objects are written as-is.
- Numbers are auto-stringified to their decimal form (for example `3.14`).
- Booleans are auto-stringified to `"1"` (true) or `"0"` (false).
- Any other type (lists, sets, tuples, etc.) is rejected.

~> **Note:** Numbers and booleans decode back as strings via `vdfdecode`, since VDF leaves are untyped. Only nested objects and strings survive a round trip unchanged.

**Common uses:** generating Steam workshop or app config files, dedicated-server configs for Source-engine games, or any other artifact where downstream Valve tooling expects VDF input.