Returns the hex-encoded `algorithm` digest of the first `CERTIFICATE` block's DER bytes, the same value `openssl x509 -fingerprint -<algorithm>` produces, minus the colon separators between byte pairs. Handy for pinning a certificate or comparing it against a known-good digest.

`algorithm` is one of:

- `"sha1"`: supported only for compatibility with older systems
- `"sha256"`: the standard choice in 2026
- `"sha384"`
- `"sha512"`

(`sha224` is not commonly used for fingerprints and is omitted.)

~> **Note:** Bundle order matters. Like `x509_inspect`, this hashes the *first* `CERTIFICATE` block in the input, which is the leaf in a conventionally-ordered fullchain.pem but not in an intermediate-first bundle. Pre-split the bundle if you need to fingerprint a specific certificate.