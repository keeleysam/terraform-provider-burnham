Decodes a PEM-encoded X.509 certificate into a structured object you can assert on at plan time (expiry comparisons, SAN coverage, key usage) without shelling out to `openssl`.

Parses the first `CERTIFICATE` block in `pem` and returns a fixed-shape object:

- `subject`, `issuer`: RFC 2253/4514-style distinguished-name strings
- `serial_number`: decimal
- `not_before`, `not_after`: RFC 3339 timestamps
- `signature_algorithm`: e.g. `SHA256-RSA`, `ECDSA-SHA256`, `Ed25519`
- `public_key_algorithm`: e.g. `RSA`, `ECDSA`, `Ed25519`
- `is_ca`: bool, true when BasicConstraints CA flag is set
- `key_usage`: list of name strings drawn from RFC 5280 KeyUsage
- `ext_key_usage`: list of name strings drawn from RFC 5280 ExtendedKeyUsage
- `dns_names`, `email_addresses`, `ip_addresses`, `uris`: Subject Alternative Names by category

Non-CERTIFICATE PEM blocks (e.g. private keys, CSRs) before the cert are skipped, so a mixed `chain.pem` works without preprocessing.

~> **Note:** Bundle order matters. When `pem` contains multiple `CERTIFICATE` blocks (a fullchain.pem, a CMS signature, etc.) this returns the *first* one, not the leaf. The leaf-first ordering is the convention for fullchain.pem and ACME-issued bundles, but a reordered or intermediate-first bundle silently inspects a non-leaf certificate. If your input could be reordered, split the bundle upstream and pass only the leaf.

~> **Note:** This function reads structure, not trust. It does not verify the certificate's signature, validity window, or chain to any trusted root: a self-signed, expired, or revoked blob parses just fine. Don't make security decisions on the result without a separate signing or trust-validation step.

Errors when the input contains no CERTIFICATE block or the certificate fails to parse.