# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- New family **compression**: `base64zopfli` (RFC 1952 gzip via [Zopfli](https://github.com/google/zopfli)'s iterative DEFLATE encoder — a drop-in, tighter replacement for the built-in `base64gzip`; gzip header pinned to `MTIME=0` / `XFL=2` / `OS=255` for deterministic, portable output, with an optional `{ iterations }` knob, default 15) and `base64brotli` (RFC 7932 Brotli, ~8–10% smaller than `base64gzip` on text, with optional `{ quality, lgwin }`, defaults 11 / 22). Both pure-Go (`CGO_ENABLED=0`, via `foobaz/go-zopfli` and `andybalholm/brotli`) and deterministic for plan stability. The RFC 7932 §10 `mode` hint is intentionally not exposed — the pure-Go encoder reads it only to special-case `font`, so `text`/`generic` are byte-identical and a `mode` option would be a no-op.
- New family **cryptography**: `hmac` (RFC 2104), `hkdf` (RFC 5869), `pem_decode` (RFC 7468), `x509_inspect`, `x509_fingerprint`, `csr_inspect` (PKCS#10), `asn1_decode` (BER/DER walk), plus a deterministic signing pipeline supporting both ECDSA P-256 and Ed25519: `ecdsa_p256_key_from_seed` (HKDF-SHA256 seed → secp256r1 scalar → PKCS#8 PEM), `ed25519_key_from_seed` (HKDF-SHA256 seed → 32-byte Ed25519 seed → PKCS#8 PEM per RFC 8032 §5.1.5), `x509_self_sign` (RFC 5280 v3 self-signed cert — ECDSA via RFC 6979 deterministic `k`, Ed25519 via PureEdDSA naturally deterministic; serial bounded to 20 octets per §4.1.2.2, UTCTime/GeneralizedTime split at 2050 per §4.1.2.5, BasicConstraints critical with cA=FALSE per §4.2.1.9, Ed25519 algorithm encoding per RFC 8410), and `pkcs7_sign` (RFC 5652 SignedData ContentInfo, encapsulated `id-data`, no signed attributes per §5.3, embedded cert, byte-stable across runs — ECDSA signs SHA-256(data), Ed25519 signs raw data with `id-sha512` digest OID per RFC 8419 §3). The chain composes but `pkcs7_sign` also accepts caller-supplied real-world identities for the CA-issued case and rejects mismatched key/cert pairs at call time rather than producing silently-unverifiable output. macOS configuration-profile signing requires ECDSA P-256; Ed25519 is for non-Apple consumers (OpenSSL `cms`, container signing).
- New family **identifiers**: deterministic `uuid_v5` (RFC 9562 §5.5), deterministic `uuid_v7` (RFC 9562 §5.7), `uuid_inspect`, deterministic `nanoid`, deterministic `petname`.
- New family **text**: `unicode_normalize` (UAX #15 NFC/NFD/NFKC/NFKD), `slugify`, `levenshtein`, `wrap`, `cowsay`, `qr_ascii`.
- New family **geographic**: `geohash_encode`, `geohash_decode`, `pluscode_encode`, `pluscode_decode` (Open Location Code).
- Numerics expansion: `mean`, `median`, `mode`, `percentile`, `variance`, `stddev`, `clamp`, `mod_floor`.
- Network expansion: `pigeon_throughput` — RFC 1149 / RFC 2549 IP-over-Avian-Carriers throughput calculator.
- Network expansion: `ip_idunno_encode` / `ip_idunno_decode` — RFC 8771 Internationalized Deliberately Unreadable Network Notation. Dual-stack, reaches the §4.1 Minimum Confusion Level on every output, and reproduces the §5 worked example bit-for-bit.
- `cmd/gendoctemplates` writes per-function `subcategory:` headers so the registry sidebar groups functions by family.

### Changed
- Provider Schema description rewritten to enumerate all nine families.
- README adds a "Tagged-value helpers" sub-table covering `plistdata`/`plistdate`/`plistreal` and `regbinary`/`regdword`/`regexpandsz`/`regmulti`/`regqword`, which were registered but not previously documented in the README.
- All `MarkdownDescription` strings across `dataformat/`, `network/`, and `transform/` collapsed to single literals (no mid-paragraph `+` joins).
- `geohash_decode` parameter renamed `hash` → `code` to match `pluscode_decode` and the broader geographic-family naming.
- `asn1_decode`'s `children` field is now `dynamic` (a tuple at runtime) instead of `list(dynamic)`. Tuples in HCL still accept `children[i]`, `length(children)`, and `[for x in children : ...]` — but `tolist(children)` and any type-assertion that explicitly demanded `list(...)` will fail. This was forced by the heterogeneous-children panic fix (see Fixed); homogeneous-children HCL that relied on list typing should `[for x in children : x]` to coerce.
- HMAC and HKDF docstrings now share a single `hclByteHandlingGotcha` helper so the byte-handling explanation cannot drift between the two functions.
- CI now runs the full acceptance suite with `TF_ACC=1` and `-race`.

### Fixed
- `asn1_decode` no longer panics when decoding ASN.1 structures with heterogeneous children (e.g. CMS SignedData's SET children: SEQUENCE, OCTET STRING, [0]-tagged blobs). The decoder's `children` field is now a `Dynamic`-typed tuple instead of `list(dynamic)`, sidestepping cty's "inconsistent value types in ListVal" panic; HCL accessor syntax (`children[0]`, `length(children)`) is unaffected.
- `nanoid` no longer panics on a 256-codepoint alphabet (`byte(256)` overflow → `% 0` divide-by-zero); modulus arithmetic is now in `int`.
- `regdword` / `regqword` now reject negative, fractional, and out-of-range inputs explicitly. Previously `(*big.Float).Uint64()` silently saturated negatives to `0` and overflow to `MaxUint*`.
- `geohash_encode` rejects exact `lat == 90` / `lon == 180` (upstream encoder wrapped these to the opposite corner) and `geohash_decode` shrinks the corner-cell bbox edges below the wrap threshold so feeding `lat_max` / `lon_max` back into the encoder round-trips into the same cell.
- ASN.1 decoder now bounds total node count (≤ 100,000), input length (≤ 8 MiB), and recursion depth (≤ 64 levels) to defeat adversarial blobs that would otherwise OOM the provider.
- CBOR decoder sets `MaxNestedLevels`, `MaxArrayElements`, `MaxMapPairs` and CBOR/MessagePack/VDF/KDL all bound input length to 16 MiB; `convert.go`'s `goToTerraformValue` / `terraformValueToGo` now cap recursion at 1024 levels.
- Cowsay rejects non-printable runes in `eyes` / `tongue` (no more ANSI escape smuggling) and caps the input message at 64 KiB.
- Pluscode docstring example output corrected to the canonical `849VQHFJ+X6` for `(37.7749, -122.4194, 10)`.
- Provider Schema no longer mis-credits Pi to Chudnovsky; the plan-time path uses an embedded DPD-packed table.

## Earlier releases

For releases prior to the introduction of this changelog (`v0.1.0` through
`v0.1.5`), see the [GitHub Releases page](https://github.com/keeleysam/terraform-provider-burnham/releases)
— each release was published with auto-generated notes derived from the
commit history.
