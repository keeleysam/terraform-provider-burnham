<!-- Edit here: this is the MarkdownDescription source for the burnham base64decode function. docs/functions/base64decode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Decodes base64 to its bytes, returned as a string of those raw bytes.

Deliberately lenient, so it is a friction-free superset of Terraform's built-in `base64decode` (which rejects URL-safe input) and round-trips anything `base64encode` produces regardless of its options:

- Accepts both the standard and the URL-safe (§5) alphabets.
- Tolerates missing `=` padding.
- Ignores ASCII whitespace.

-> **Note:** The result is a byte string. For binary that isn't valid UTF-8 you will usually feed it into another function rather than printing it.

```
base64decode("SGVsbG8")   # unpadded, standard alphabet
→ "Hello"
```