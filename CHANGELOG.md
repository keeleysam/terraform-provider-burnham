# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- New family **cryptography**: `hmac` (RFC 2104), `hkdf` (RFC 5869), `pem_decode` (RFC 7468), `x509_inspect`, `x509_fingerprint`, `csr_inspect` (PKCS#10), `asn1_decode` (BER/DER walk).
- New family **identifiers**: deterministic `uuid_v5` (RFC 9562 §5.5), deterministic `uuid_v7` (RFC 9562 §5.7), `uuid_inspect`, deterministic `nanoid`, deterministic `petname`.
- New family **text**: `unicode_normalize` (UAX #15 NFC/NFD/NFKC/NFKD), `slugify`, `levenshtein`, `wrap`, `cowsay`, `qr_ascii`.
- New family **geographic**: `geohash_encode`, `geohash_decode`, `pluscode_encode`, `pluscode_decode` (Open Location Code).
- Numerics expansion: `mean`, `median`, `mode`, `percentile`, `variance`, `stddev`, `clamp`, `mod_floor`.
- Network expansion: `pigeon_throughput` — RFC 1149 / RFC 2549 IP-over-Avian-Carriers throughput calculator.
- `cmd/gendoctemplates` writes per-function `subcategory:` headers so the registry sidebar groups functions by family.

### Changed
- Provider Schema description rewritten to enumerate all eight families.
- README adds a "Tagged-value helpers" sub-table covering `plistdata`/`plistdate`/`plistreal` and `regbinary`/`regdword`/`regexpandsz`/`regmulti`/`regqword`, which were registered but not previously documented in the README.
- All `MarkdownDescription` strings across `dataformat/`, `network/`, and `transform/` collapsed to single literals (no mid-paragraph `+` joins).
- `geohash_decode` parameter renamed `hash` → `code` to match `pluscode_decode` and the broader geographic-family naming.
- HMAC and HKDF docstrings now share a single `hclByteHandlingGotcha` helper so the byte-handling explanation cannot drift between the two functions.
- CI now runs the full acceptance suite with `TF_ACC=1` and `-race`.

### Fixed
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
