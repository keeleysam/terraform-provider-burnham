<!-- Edit here: this is the MarkdownDescription source for the burnham hexencode function. docs/functions/hexencode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Encodes the input's bytes as a lowercase hexadecimal string (two hex digits per byte).

The input is taken as raw bytes, the literal UTF-8 bytes of the string HCL hands the function. To hex-encode bytes you already hold as base64, pass `base64decode(var.x)`.

```
hexencode("Hi")
→ "4869"
```