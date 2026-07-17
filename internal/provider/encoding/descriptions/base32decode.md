<!-- Edit here: this is the MarkdownDescription source for the burnham base32decode function. docs/functions/base32decode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Decodes base32 to its bytes, returned as a string of those raw bytes.

Decoding is lenient, so a TOTP secret pasted in any case, padded or not, decodes cleanly:

- The input is uppercased (base32 is case-insensitive in practice).
- ASCII whitespace is ignored.
- Missing `=` padding is tolerated.

~> **Note:** Unlike `base64decode`, the alphabet cannot be auto-detected: the standard (`A–Z2–7`) and extended-hex (`0–9A–V`) alphabets overlap, so an ambiguous string could be either. Pass `{ hex_alphabet = true }` to decode the hex alphabet; the default is standard.

-> **Note:** The result is a byte string. For binary that isn't valid UTF-8 you will usually feed it into another function (for example `hmac("sha1", base32decode(var.totp_secret), …)`) rather than printing it.

```
base32decode("mzxw6YTBoi")   # unpadded, mixed case, fine
→ "foobar"
```