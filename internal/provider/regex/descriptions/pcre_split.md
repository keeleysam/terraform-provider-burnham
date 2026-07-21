Splits `str` around every non-overlapping match of `pattern` and returns the pieces as a list. The matched separators are removed; the text between (and before/after) them is kept, including empty strings where separators are adjacent or at the ends.

Uses PCRE syntax via [fancy-regex](https://github.com/fancy-regex/fancy-regex), so the separator pattern can use lookaround and other PCRE features. An invalid pattern is a plan-time error.

Runs as WebAssembly under a pure-Go runtime: CGO-free, deterministic, with bounded backtracking.
