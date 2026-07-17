<!-- Edit here: this is the MarkdownDescription source for the burnham parse_kv function. docs/functions/parse_kv.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Parses a delimited key/value string into a `map(string)`. This replaces the fragile HCL idiom `{ for p in split(",", s) : split("=", p)[0] => split("=", p)[1] }`, which breaks the moment a value contains an `=`, carries surrounding whitespace, or is quoted to protect a literal separator.

```
parse_kv("a=1,b=2")            → { a = "1", b = "2" }
parse_kv("url=https://x?a=b")  → { url = "https://x?a=b" }
parse_kv("a=\"x,y\",b=2")      → { a = "x,y", b = "2" }
```

Options object (all optional):

- `pair_sep` (string): separator between pairs. Default `","`.
- `kv_sep` (string): separator between a key and its value. Default `"="`.
- `trim` (bool): trim leading and trailing ASCII whitespace around each key and value. Default `true`.
- `unquote` (bool): parse quote-aware. A key or value wrapped in matching double quotes (`"..."`) or single quotes (`'...'`) is unwrapped, and any `pair_sep` or `kv_sep` inside the quotes is treated as literal, not a separator. With `unquote = false`, quotes are literal characters and splitting is purely on the separators. Default `true`.

Behavior:

- Each pair is split on the **first** `kv_sep` only, so a value may itself contain `kv_sep`: `parse_kv("a=b=c")` returns `{ a = "b=c" }`.
- Empty segments are skipped, so a trailing separator or a doubled separator is ignored: `parse_kv("a=1,,b=2,")` returns `{ a = "1", b = "2" }`.
- All values are strings; numbers and booleans are not coerced, matching what the naive HCL workaround produced.

-> **Note:** A quote counts as a wrapper only when it is the first and last character of the (trimmed) field, so `b"c"` is kept verbatim and only a fully wrapped `"..."` is unwrapped.

~> **Errors:** A pair with no `kv_sep` is an error (the offending pair is quoted in the message), and a duplicate key is an error (a `map(string)` cannot hold duplicates, so the message names the repeated key). `pair_sep` and `kv_sep` must each be non-empty and must differ from each other.