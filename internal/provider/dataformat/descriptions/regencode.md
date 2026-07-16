Encodes a Terraform object as a [Windows Registry Editor export (`.reg`) file](https://learn.microsoft.com/en-us/windows/win32/sysinfo/regedit) in Version 5 format.

The input must be a two-level map: registry key paths at the outer level, value names at the inner level. The default value uses the key name `"@"`.

Plain strings become `REG_SZ` values. Tagged objects from the constructor functions convert to their native registry types:

- `regdword()` becomes `REG_DWORD`
- `regqword()` becomes `REG_QWORD`
- `regbinary()` becomes `REG_BINARY`
- `regmulti()` becomes `REG_MULTI_SZ`
- `regexpandsz()` becomes `REG_EXPAND_SZ`

Pass an optional `comments` key in `options`, mirroring the data structure, to inject `;` comments above a registry key path (a top-level string) or above a specific value line (nest a `value name = comment` object under the key path).

**Common uses:** generating `.reg` files for endpoint management (Group Policy alternatives, app preferences, security baselines, or developer-tooling defaults) and committing them to a managed config repo.