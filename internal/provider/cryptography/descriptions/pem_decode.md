<!-- Edit here: this is the MarkdownDescription source for the burnham pem_decode function. docs/functions/pem_decode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Splits PEM-armoured text (RFC 7468) into a structured list, one entry per block, so you can pull a certificate, key, or CSR body out of a bundle and hand it to another function.

Each entry has:

- `type`: the block label between `-----BEGIN ` / `-----END ` (e.g. `"CERTIFICATE"`, `"PRIVATE KEY"`, `"CERTIFICATE REQUEST"`).
- `headers`: `map(string)` of any RFC 1421 / 7468 header lines (often empty for modern PEM).
- `base64_body`: the body, kept base64-encoded so the bytes round-trip exactly through `base64decode`. The body is the standard base64 alphabet, no line breaks.

Returns an empty list when the input contains no PEM blocks. Garbage between blocks is silently skipped, the same behaviour as `openssl` and most consumers.

~> **Note:** To bound memory, input larger than 16 MiB, or containing more than 100,000 blocks, is rejected. Both limits are far above any real bundle (a fullchain.pem is a few KiB and a handful of blocks).