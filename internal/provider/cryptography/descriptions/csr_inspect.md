<!-- Edit here: this is the MarkdownDescription source for the burnham csr_inspect function. docs/functions/csr_inspect.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Decodes a PKCS #10 certificate signing request (RFC 2986) into a structured object, useful at plan time for asserting that a CSR's subject, SANs, or signature algorithm are what you expect before handing it to a CA.

Parses the first `CERTIFICATE REQUEST` (or the legacy pre-RFC 7468 `NEW CERTIFICATE REQUEST`) block in `pem` and returns a fixed-shape object:

- `subject`: RFC 2253/4514-style distinguished-name string from the CSR
- `signature_algorithm`: e.g. `SHA256-RSA`, `ECDSA-SHA256`, `Ed25519`
- `public_key_algorithm`: e.g. `RSA`, `ECDSA`, `Ed25519`
- `dns_names`, `email_addresses`, `ip_addresses`, `uris`: Subject Alternative Names by category, taken from the CSR's requested-extensions attribute

Fields that don't exist on a CSR (serial number, validity window, key usage, BasicConstraints) are not on this object; those are set by the issuing CA when the request is approved.

~> **Note:** This function reads structure, not trust. It does not verify the CSR's self-signature. Treat the result as the *requested* attributes; the issuing CA decides what actually ends up on the certificate.

Errors when the input contains no certificate-request block or the request fails to parse.