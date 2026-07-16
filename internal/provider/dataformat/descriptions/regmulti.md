Returns a tagged object representing a `REG_MULTI_SZ` (null-separated list of strings) registry value, for use inside a `regencode` payload.

**Common uses:** registry values that are inherently lists, such as search paths, allowlist/denylist entries, or any field where the consuming Windows component expects multi-string semantics rather than a single delimited string.