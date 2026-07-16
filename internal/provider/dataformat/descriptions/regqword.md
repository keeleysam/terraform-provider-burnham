Returns a tagged object representing a `REG_QWORD` (64-bit unsigned integer) registry value, for use inside a `regencode` payload.

Pass the value as a decimal integer between `0` and `18446744073709551615`. HCL's number type (a 512-bit big.Float) carries the full range exactly. HCL doesn't accept `0x...` literals; convert to decimal manually or use `parseint("...", 16)`.

**Common uses:** large numeric values in registry-driven config, such as file size limits, byte offsets, or any integer that exceeds `REG_DWORD`'s 32-bit range.