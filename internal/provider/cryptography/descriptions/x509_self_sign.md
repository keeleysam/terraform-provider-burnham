Builds a self-signed X.509 v3 certificate from a PEM-encoded private key and returns it as PEM. Use it, together with a deterministic key, to mint a stable signing identity that does not churn in Terraform state.

Signing is deterministic on both supported key types: ECDSA P-256 uses RFC 6979 deterministic `k`, Ed25519 uses PureEdDSA (naturally deterministic per RFC 8032). Given the same `private_key_pem` and the same parameters, the output is byte-identical across runs. Paired with [`ecdsa_p256_key_from_seed`](#function-ecdsa_p256_key_from_seed) or [`ed25519_key_from_seed`](#function-ed25519_key_from_seed), the full chain from seed to key to cert is deterministic, with no random state at any step.

Certificate fields:

- **Version**: 3.
- **Serial Number**: derived from `serial` (raw bytes; interpreted big-endian, leading-byte high bit cleared so the DER-encoded length stays predictable). must be non-empty and at most 20 bytes (RFC 5280 §4.1.2.2 caps the encoded length at 20 octets); at least 8 bytes is recommended for uniqueness.
- **Issuer = Subject**: a single Common Name attribute (self-signed).
- **Validity**: as supplied, RFC 3339.
- **Basic Constraints**: critical, `CA:FALSE`.
- **Signature Algorithm**: `ecdsa-with-SHA256` for ECDSA P-256 keys, `Ed25519` ([RFC 8410](https://www.rfc-editor.org/rfc/rfc8410)) for Ed25519 keys.

Key input:

- Only ECDSA P-256 and Ed25519 keys are accepted; other key types return an error.
- PEM must contain a `PRIVATE KEY` (PKCS#8) or `EC PRIVATE KEY` (SEC1) block.

%s