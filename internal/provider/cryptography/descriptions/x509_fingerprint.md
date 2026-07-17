<!-- Edit here: this is the MarkdownDescription source for the burnham x509_fingerprint function. docs/functions/x509_fingerprint.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns the hex-encoded `algorithm` digest of the first `CERTIFICATE` block's DER bytes, the same value `openssl x509 -fingerprint -<algorithm>` produces, lowercased and with the colon separators between byte pairs removed. Handy for pinning a certificate or comparing it against a known-good digest.

`algorithm` is one of:

- `"sha1"`: supported only for compatibility with older systems
- `"sha256"`: the standard choice in 2026
- `"sha384"`
- `"sha512"`

(`sha224` is not commonly used for fingerprints and is omitted.)

~> **Note:** Bundle order matters. Like `x509_inspect`, this hashes the *first* `CERTIFICATE` block in the input, which is the leaf in a conventionally-ordered fullchain.pem but not in an intermediate-first bundle. Pre-split the bundle if you need to fingerprint a specific certificate.