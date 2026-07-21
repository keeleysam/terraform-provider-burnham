Reports whether `pattern` matches anywhere in `str`, using PCRE-style regular expressions.

Terraform core's `regex` / `regexall` and this provider's other matching use Go's RE2 engine, which is fast and linear-time but deliberately omits backreferences and lookaround. `pcre_match` is backed by [fancy-regex](https://github.com/fancy-regex/fancy-regex), so patterns can use `\1` backreferences, lookahead `(?=...)` / `(?!...)`, and lookbehind `(?<=...)` / `(?<!...)`. Set inline flags in the pattern, for example `(?i)` for case-insensitive or `(?s)` for dot-matches-newline.

Returns `true` if the pattern matches any substring, `false` otherwise. An invalid pattern is a plan-time error.

The engine is [fancy-regex](https://github.com/fancy-regex/fancy-regex) compiled to WebAssembly and run under a pure-Go runtime, so the provider stays CGO-free and results are deterministic. Backtracking is bounded, so a pathological pattern fails with an error rather than hanging the plan.
