<!-- Edit here: this is the MarkdownDescription source for the burnham promqlformat function. docs/functions/promqlformat.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Parses `query` and returns its canonical [PromQL](https://prometheus.io/docs/prometheus/latest/querying/basics/) serialization: normalized spacing and operator layout, on a single line.

Pass `{ pretty = true }` for the parser's multi-line, indented form, which wraps only long sub-expressions (nice for a long alerting expression).

The output is stable and idempotent, so two queries that differ only in whitespace canonicalize to the same string, and the default single-line output is byte-identical to what `promqlencode` produces. Two normalizations to expect:

- Label matchers within a selector are sorted alphabetically.
- PromQL `#` comments are dropped.

~> **Note:** Fails the plan on invalid input, or on a query longer than 64 KiB. Use `promqlvalidate` for a non-failing boolean check. Backed by [prometheus/prometheus](https://github.com/prometheus/prometheus)'s own parser.