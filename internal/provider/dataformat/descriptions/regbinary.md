Returns a tagged object representing a `REG_BINARY` registry value, for use inside a `regencode` payload. The input is a hex-encoded string (no separators, no `0x` prefix).

**Common uses:** binary blobs in Group Policy and app preferences, such as certificate hashes, packed structures, or pre-computed configuration payloads consumed by Windows components.