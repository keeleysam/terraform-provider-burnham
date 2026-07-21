Returns every non-overlapping match of `pattern` in `str`, as a list of the matched substrings (in order). Returns an empty list when there are no matches.

Uses PCRE syntax via [fancy-regex](https://github.com/fancy-regex/fancy-regex), so the pattern can use backreferences, lookahead/lookbehind, and inline flags such as `(?i)`, none of which Go's RE2-based `regexall` supports. An invalid pattern is a plan-time error.

Runs as WebAssembly under a pure-Go runtime: CGO-free, deterministic, with bounded backtracking.
