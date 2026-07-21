Replaces every non-overlapping match of `pattern` in `str` with `replacement`, and returns the result.

The replacement supports backreferences to capture groups: `$1` (or `${1}`) for numbered groups and `${name}` for named groups `(?<name>...)`. Write `$$` for a literal dollar sign. Backreferences in the replacement use this `$`-syntax, not the PCRE `\1` form: a literal `\1` in the replacement stays literal, and a reference to a group that did not participate expands to the empty string.

In an HCL double-quoted string, `${...}` is Terraform interpolation, so to pass a named backreference through to the replacement you must escape it as `$${name}` (which yields the literal `${name}`). The numbered `$1` form needs no escaping.

The argument order is `(pattern, str, replacement)` with the pattern first, matching the other `pcre_*` functions and Terraform core's `regex`, rather than the subject-first order of core's `replace`.

Uses PCRE syntax via [fancy-regex](https://github.com/fancy-regex/fancy-regex), so the pattern can use backreferences and lookaround, unlike Terraform core's RE2-based `replace` with a regex. An invalid pattern is a plan-time error.

Runs as WebAssembly under a pure-Go runtime: CGO-free, deterministic, with bounded backtracking.
