<!-- Edit here: this is the MarkdownDescription source for the burnham regexpandsz function. docs/functions/regexpandsz.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns a tagged object representing a `REG_EXPAND_SZ` registry value, for use inside a `regencode` payload. `REG_EXPAND_SZ` differs from `REG_SZ` in that the consuming Windows component expands `%VARIABLE%` references at lookup time.

**Common uses:** path values that must adapt per-user or per-machine (`%APPDATA%`, `%SystemRoot%`, `%USERPROFILE%`), or any registry-driven config that needs to substitute environment variables when read.