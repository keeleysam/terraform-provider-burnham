<!-- Edit here: this is the MarkdownDescription source for the burnham hexdecode function. docs/functions/hexdecode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Decodes a hexadecimal string to its bytes, returned as a string of those raw bytes.

Decoding is lenient:

- Upper- and lower-case digits are both accepted.
- ASCII whitespace is ignored, so a spaced or line-wrapped dump decodes cleanly.

-> **Note:** The result is a byte string. For binary that isn't valid UTF-8 you will usually feed it straight into another function (for example `hmac("sha256", hexdecode(var.key_hex), var.msg)`) rather than printing it.

```
hexdecode("4869")
→ "Hi"
```