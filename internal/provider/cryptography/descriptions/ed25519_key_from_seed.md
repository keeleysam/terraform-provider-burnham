<!-- Edit here: this is the MarkdownDescription source for the burnham ed25519_key_from_seed function. docs/functions/ed25519_key_from_seed.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Derives an Ed25519 private key deterministically from `seed` and returns it as PEM PKCS#8. The same `seed` always produces the same key, so you get a stable signing identity from a checked-in secret or input artefact rather than a randomly generated key stored in state.

`seed` is stretched to 32 bytes with HKDF-SHA256 (info string `%q`) and used as the Ed25519 private-key seed per [RFC 8032 §5.1.5](https://www.rfc-editor.org/rfc/rfc8032#section-5.1.5).

Pair with [`x509_self_sign`](#function-x509_self_sign) and [`pkcs7_sign`](#function-pkcs7_sign): both accept either ECDSA P-256 or Ed25519 keys and dispatch the right signing algorithm on the key type.

Ed25519 is deterministic by spec (the per-signature nonce is derived by SHA-512 over a private-key prefix and the message, per RFC 8032), so unlike the ECDSA signing path there is no RFC 6979 wrapper involved; the stdlib `crypto/ed25519` signer is already byte-stable.

~> **Note:** Ed25519 in CMS / X.509 is supported by OpenSSL and the rest of the modern PKI ecosystem ([RFC 8032](https://www.rfc-editor.org/rfc/rfc8032), [RFC 8410](https://www.rfc-editor.org/rfc/rfc8410), [RFC 8419](https://www.rfc-editor.org/rfc/rfc8419)), but is **not accepted** by Apple's macOS configuration-profile installer at the keychain-import layer as of macOS 26.5: signed `.mobileconfig` files using Ed25519 fail at install time. For Apple configuration profiles use [`ecdsa_p256_key_from_seed`](#function-ecdsa_p256_key_from_seed) instead. Ed25519 is the better choice when the signature consumer is anything else (OpenSSL `cms`, GPG-replacement workflows, container signing, internal tooling).

%s