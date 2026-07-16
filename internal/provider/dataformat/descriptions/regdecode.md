Parses a [Windows Registry Editor export (`.reg`) file](https://learn.microsoft.com/en-us/windows/win32/sysinfo/regedit) into a Terraform object.

Both header versions are auto-detected:

- Version 4: `REGEDIT4` (ANSI)
- Version 5: `Windows Registry Editor Version 5.00` (UTF-16-as-text)

The result is a two-level map: registry key paths at the outer level, value names at the inner level. The default value (`@`) is keyed as `"@"`.

`REG_SZ` values become plain strings. Every other type decodes as a tagged object with `__reg_type` and `value` keys, round-trippable through `regencode`:

- `REG_DWORD`
- `REG_QWORD`
- `REG_BINARY`
- `REG_MULTI_SZ`
- `REG_EXPAND_SZ`
- `REG_NONE` (its `value` is a hex-encoded string)

**Common uses:** importing existing `.reg` exports from a reference machine, normalizing them into a typed Terraform value, or staging registry policy snapshots for diff review before redeployment.