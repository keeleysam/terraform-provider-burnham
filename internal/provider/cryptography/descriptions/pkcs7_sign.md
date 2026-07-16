CMS / PKCS#7 signs `data` with an ECDSA P-256 or Ed25519 identity and returns base64-encoded DER. Use it as the on-the-wire signing primitive for signed configuration profiles, CMS payloads, and similar artefacts that must be byte-stable across Terraform runs.

It produces an RFC 5652 SignedData ContentInfo that carries `data` as its encapsulated `id-data` content, with the signer cert embedded and no signed attributes (RFC 5652 §5.3 permits omitting `signedAttrs` when the encapsulated content type is `id-data`). Decode the result with `base64decode(...)` or feed it straight into `local_file.content_base64`.

~> **Note:** Output is deterministic by construction: identical `(data, private_key_pem, cert_pem)` always yields the same DER bytes.

Signing dispatches on the key type:

- **ECDSA P-256**: RFC 6979 deterministic ECDSA-with-SHA256.
- **Ed25519**: PureEdDSA ([RFC 8419](https://www.rfc-editor.org/rfc/rfc8419) §3.1, message signed directly, no pre-hash). The SignerInfo's `digestAlgorithm` is set to `id-sha512` per RFC 8419 §3 (vestigial under no-signed-attrs, but required to be present in the SignedData `digestAlgorithms` SET).

Input requirements:

- `cert_pem`'s public key must match `private_key_pem`. The match is checked at call time, so a mismatch is rejected rather than producing an unverifiable signature.
- No chain validation is performed. This is the on-the-wire signing primitive, not a PKI workflow.
- `data` must be 1 byte to %d bytes (%d MiB).

The emitted SignedData has `version: 1` and `SignerInfo.version: 1` per RFC 5652 §5.1 / §5.3 (encapsulated content is `id-data`, signer identified by `issuerAndSerialNumber`, no version-3-certificate or OtherCertificateFormat children).

~> **Note:** Apple configuration-profile signing uses the ECDSA P-256 variant of this shape, and Jamf passes signed profiles through unchanged. Apple's macOS profile installer rejects Ed25519-signed mobileconfigs at the keychain-import layer as of macOS 26.5, so use Ed25519 here only for non-Apple consumers (OpenSSL `cms`, container signing, internal tooling).

-> **Note:** This function intentionally emits the no-signed-attributes flavour. If you need the more typical CMS shape *with* signed attributes (and the resulting `signingTime` non-determinism), use a different library.

%s