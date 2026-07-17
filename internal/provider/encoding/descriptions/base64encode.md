<!-- Edit here: this is the MarkdownDescription source for the burnham base64encode function. docs/functions/base64encode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Base64-encodes the input's bytes per [RFC 4648](https://www.rfc-editor.org/rfc/rfc4648). With no options it produces standard, padded base64, identical to Terraform's built-in `base64encode`. The optional object selects the variant:

- `url_safe` (bool, default `false`): use the URL- and filename-safe alphabet (§5: `-` and `_` instead of `+` and `/`), as used by JWT/JOSE and OAuth PKCE.
- `padding` (bool, default `true`): emit `=` padding. Set `false` for the raw, unpadded form some APIs require.

The input is taken as raw bytes (the literal UTF-8 bytes of the string); to encode bytes held as hex, pass `hexdecode(var.x)`.

```
base64encode("Hello")                          → "SGVsbG8="
base64encode(var.token, { url_safe = true, padding = false })
```