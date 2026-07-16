Percent-encodes a string for use in a URL. With no options it uses `mode = "query"` (`application/x-www-form-urlencoded`, encoding a space as `+`), which is byte-identical to Terraform's built-in `urlencode`. The optional `mode` selects where the value is going:

- `"query"` (default): form encoding; space → `+`. For `a=b&c=d` query strings.
- `"path"`: [RFC 3986](https://www.rfc-editor.org/rfc/rfc3986) path segment; space → `%20`, `/` escaped, `+` left literal. For building path components.
- `"component"`: strict; everything except the unreserved set `A-Za-z0-9-_.~` is escaped, space → `%20`. For a value that must be safe in *any* URL position.

Core's `urlencode` only does the `query` form, whose space → `+` is wrong inside a path; `path`/`component` fix that.

```
urlencode("a b/c")                      → "a+b%2Fc"
urlencode("a b/c", { mode = "path" })   → "a%20b%2Fc"
```