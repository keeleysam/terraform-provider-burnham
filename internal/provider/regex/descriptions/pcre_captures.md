Returns the capture groups of the first PCRE match of `pattern` in `str`, as a map of group name to matched text.

The map is keyed by both the numbered groups (`"0"` is the whole match, `"1"`, `"2"`, ... are the parenthesized groups) and any named groups (`(?<name>...)`), so you can read a group either way. Groups that did not participate in the match are omitted. If the pattern does not match at all, the result is an empty map.

Uses PCRE syntax via [fancy-regex](https://github.com/fancy-regex/fancy-regex) (backreferences, lookaround, inline flags like `(?i)`), which Go's RE2-based engine cannot express. An invalid pattern is a plan-time error.

Runs as WebAssembly under a pure-Go runtime: CGO-free, deterministic, with bounded backtracking.
