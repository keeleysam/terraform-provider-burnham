Returns a tagged object representing a `REG_DWORD` (32-bit unsigned integer) registry value, for use inside a `regencode` payload.

Pass the value as a decimal integer between `0` and `4294967295`. HCL doesn't accept `0x...` literals; convert to decimal manually or use `parseint("01020304", 16)`.

**Common uses:** typed registry values in Group Policy / endpoint config, such as feature flags, integer thresholds, and status fields that must be `REG_DWORD` rather than `REG_SZ`.