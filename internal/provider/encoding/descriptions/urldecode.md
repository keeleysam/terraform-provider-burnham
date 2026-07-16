Decodes a percent-encoded string, the function Terraform core is missing entirely. `%XX` escapes are decoded in every mode; the `mode` only controls how `+` is treated, because `+` is ambiguous (a space in a query string, a literal `+` in a path):

- `"query"` (default): form semantics; `+` → space (and `%2B` → `+`). The inverse of `urlencode`'s default.
- `"path"` / `"component"`: `+` is left literal; only `%XX` is decoded.

The result is a byte string; for input that decodes to non-UTF-8 bytes you will usually feed it into another function rather than printing it.

```
urldecode("a+b%2Fc")                      → "a b/c"
urldecode("1+1", { mode = "path" })       → "1+1"
```