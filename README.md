# Terraform Provider Burnham

<p align="center">
  <img src="assets/logo.svg" alt="Burnham" width="300" height="300">
</p>

> *"Aim high in hope and work, remembering that a noble, logical diagram once recorded will never die."*
> — Daniel Burnham

In 1909, Daniel Burnham published the [Plan of Chicago](https://en.wikipedia.org/wiki/Burnham_Plan_of_Chicago) — a comprehensive blueprint that transformed a sprawling, chaotic city into something coherent and enduring. He believed that good planning wasn't just about what you build, but about making the plan itself clear, readable, and maintainable for generations to come.

Terraform plans deserve the same treatment. But today, when your Terraform needs to work with structured data formats like property lists or human-edited JSON, do real arithmetic on IP address space, or apply environment overlays to a base manifest, you're stuck with workarounds. Shelling out to external tools, embedding raw strings, pasting opaque expressions that obscure what the plan is actually doing.

Burnham fixes this. It's a pure function provider — no resources, no data sources, no API calls — that fills the operations Terraform's expression language can't handle cleanly on its own. Structured data formats and network arithmetic at the foundation; query and patch over decoded values; deterministic identifiers, text manipulation, certificate inspection, and geographic encoding alongside; and a small numerics library where RFC-faithful curiosities live next to plain statistics.

Your configuration profiles, ACL policies, and structured documents become first-class citizens in your Terraform plans, not opaque blobs passed through `file()` and hoped for the best. Your network plans show set arithmetic on CIDRs in plain HCL instead of `templatefile()`-driven Python preprocessors. Your manifest overlays apply RFC 7396 merge patches in one expression rather than a chain of `merge()` and `try()` calls. Your TLS certificates surface their expiry, SANs, and fingerprints as structured fields instead of opaque base64 blobs.

The result is Terraform code that reads like a blueprint — clear, logical, and built to last.

Burnham is organized into eight families of functions:

- **[Structured Data Functions](#structured-data-functions)** — encode/decode for JSON (pretty), HuJSON, plist, INI, CSV, YAML, .reg, VDF, KDL, NDJSON, MessagePack, CBOR, dotenv, Java .properties, Apple .strings, and general HCL.
- **[Networking Functions](#networking-functions)** — CIDR set operations, queries, IP arithmetic, NAT64 (RFC 6052), NPTv6 (RFC 6296), IPAM helpers, and a faithful RFC 1149 / RFC 2549 (IP over Avian Carriers) throughput calculator.
- **[Query and Patch Functions](#query-and-patch-functions)** — JMESPath, JSONPath (RFC 9535), JSON Patch (RFC 6902), and JSON Merge Patch (RFC 7396) over decoded structures.
- **[Numerics Functions](#numerics-functions)** — RFC 3091 (Pi Digit Generation Protocol), statistics, and small math helpers.
- **[Identifiers Functions](#identifiers-functions)** — deterministic UUIDs (v5, v7), Nano ID, and petname.
- **[Text Functions](#text-functions)** — Unicode normalization, transliterating slugify, Levenshtein distance, word-wrap, cowsay, ASCII QR.
- **[Cryptography Functions](#cryptography-functions)** — HMAC (RFC 2104), HKDF (RFC 5869), PEM block decoding, X.509 / CSR inspection and fingerprinting, generic ASN.1 BER/DER decoding.
- **[Geographic Functions](#geographic-functions)** — geohash and Open Location Code (Plus codes), encode and decode.

## Structured Data Functions

| Format | Encode | Decode | Notes |
|--------|--------|--------|-------|
| Apple .strings | `applestringsencode` | `applestringsdecode` | Localization files. UTF-8 / UTF-16 BOM auto-detect on decode |
| Apple Property List | `plistencode` | `plistdecode` | XML (with comments), binary, and OpenStep formats |
| CBOR | `cborencode` | `cbordecode` | RFC 8949, Core Deterministic Encoding; base64-wrapped on the HCL side |
| CSV | `csvencode` | — | Terraform has `csvdecode` built-in |
| dotenv (.env) | `dotenvencode` | `dotenvdecode` | godotenv flavor: `KEY=value`, `"`/`'` quoting, `${VAR}` interpolation |
| HCL (general) | `hclencode` | `hcldecode` | Attribute-only HCL documents; for `.tfvars` use the built-in `provider::terraform::*` |
| HuJSON / JWCC | `hujsonencode` | `hujsondecode` | JSON with comments and trailing commas |
| INI | `iniencode` | `inidecode` | Standard `[section]` / `key = value` files |
| Java .properties | `javapropertiesencode` | `javapropertiesdecode` | `=`/`:`/whitespace separators, line continuation, `\uXXXX` escapes |
| JSON (pretty-printed) | `jsonencode` | — | Terraform has `jsondecode` built-in |
| KDL | `kdlencode` | `kdldecode` | Modern document language, v1 and v2 |
| MessagePack | `msgpackencode` | `msgpackdecode` | Binary format ([msgpack.org spec](https://github.com/msgpack/msgpack/blob/master/spec.md)); base64-wrapped on the HCL side |
| NDJSON / JSON Lines | `ndjsonencode` | `ndjsondecode` | One JSON value per line, trailing newline |
| TOML | — | — | Use [Tobotimus/toml](https://registry.terraform.io/providers/Tobotimus/toml) instead |
| Valve VDF | `vdfencode` | `vdfdecode` | Steam/Source engine config format |
| Windows .reg | `regencode` | `regdecode` | Registry Editor export format with typed values and comments |
| YAML | `yamlencode` | — | Block style, literal scalars, comments. Terraform has `yamldecode` built-in |

### Tagged-value helpers (consumed inside `regencode` / `plistencode`)

`REG_*` and `<plist>` documents carry typed values that aren't directly representable in HCL — `REG_DWORD` is a 32-bit unsigned integer, `<data>` is base64-wrapped binary, and so on. These constructor functions return tagged objects that the matching encoder picks up and emits with the correct type tag. Use them inline inside the encoder's input map; they're meaningless outside that context.

| Function | Returns | Used by |
|---|---|---|
| `plistdata(b64)` | tagged `data` | `plistencode` — emits as `<data>...</data>` |
| `plistdate(rfc3339)` | tagged `date` | `plistencode` — emits as `<date>...</date>` |
| `plistreal(number)` | tagged `real` | `plistencode` — emits as `<real>...</real>` (whole-number floats would otherwise round-trip as `<integer>`) |
| `regbinary(hex)` | tagged `REG_BINARY` | `regencode` — input is a hex string |
| `regdword(uint32)` | tagged `REG_DWORD` | `regencode` — accepts `[0, 4294967295]` |
| `regexpandsz(string)` | tagged `REG_EXPAND_SZ` | `regencode` — Windows expands `%VAR%` references at lookup |
| `regmulti(list(string))` | tagged `REG_MULTI_SZ` | `regencode` — null-separated list of strings |
| `regqword(uint64)` | tagged `REG_QWORD` | `regencode` — accepts `[0, 18446744073709551615]` |

Per-function documentation — including parameters, options, and return values — lives under [`docs/functions/`](docs/functions/) and on [registry.terraform.io](https://registry.terraform.io/providers/keeleysam/burnham/latest/docs). The pages there are auto-generated from the function metadata in source, so they always match the latest published version.

## Networking Functions

Pure functions for IP/CIDR work that HCL alone can't express: set arithmetic on address space, normalizing mixed IPv4/IPv6 inputs, NAT64 / NPTv6 translation, IPAM-style allocation, and one exhaustively faithful implementation of RFC 1149 / RFC 2549 (IP over Avian Carriers). All functions are pure (no network calls, no state) and evaluate at plan time. Uses [`go4.org/netipx`](https://pkg.go.dev/go4.org/netipx) for set operations, prefix aggregation, and range conversion.

The **Backed by** column matters for understanding where bugs live. Functions backed by `netipx` or `net/netip` are thin parsing wrappers — if the logic is wrong, it's almost certainly in the upstream library, not here. Functions with custom or RFC-derived implementations are where this provider adds real logic of its own.

### Standard CIDR / IP / NAT64 / NPTv6

| Function | Signature | Returns | Backed by |
|---|---|---|---|
| `cidr_contains` | `(cidr string, other string)` | `bool` | netipx `IPSet.ContainsPrefix/Contains` |
| `cidr_enumerate` | `(cidr string, newbits number)` | `list(string)` | custom iterator |
| `cidr_expand` | `(cidr string)` | `list(string)` | custom iterator |
| `cidr_filter_version` | `(cidrs list(string), version number)` | `list(string)` | custom |
| `cidr_find_free` | `(pool list(string), used list(string), prefix_len number)` | `string\|null` | netipx `IPSet.RemoveFreePrefix()` |
| `cidr_first_ip` | `(cidr string)` | `string` | `net/netip` `Prefix.Addr` |
| `cidr_host_count` | `(cidr string)` | `number` | custom (bit math) |
| `cidr_intersect` | `(a list(string), b list(string))` | `list(string)` | netipx `IPSetBuilder.Intersect` |
| `cidr_is_private` | `(cidr string)` | `bool` | custom (RFC 1918/4193/6598 table) |
| `cidr_last_ip` | `(cidr string)` | `string` | netipx `RangeOfPrefix().To()` |
| `cidr_merge` | `(cidrs list(string))` | `list(string)` | netipx `IPSetBuilder` + `Prefixes()` |
| `cidr_overlaps` | `(a string, b string)` | `bool` | `net/netip` `Prefix.Overlaps` |
| `cidr_prefix_length` | `(cidr string)` | `number` | `net/netip` `Prefix.Bits` |
| `cidr_subtract` | `(input list(string), exclude list(string))` | `list(string)` | netipx `IPSetBuilder` |
| `cidr_usable_host_count` | `(cidr string)` | `number` | custom (bit math + RFC 3021) |
| `cidr_version` | `(cidr string)` | `number` | `net/netip` `Prefix.Addr.Is4` |
| `cidr_wildcard` | `(cidr string)` | `string` | custom (bit math, IPv4 only) |
| `cidrs_are_disjoint` | `(cidrs list(string))` | `bool` | netipx `IPSet.OverlapsPrefix` |
| `cidrs_containing_ip` | `(ip string, cidrs list(string))` | `list(string)` | `net/netip` `Prefix.Contains` |
| `cidrs_overlap_any` | `(a list(string), b list(string))` | `bool` | netipx `IPSet.OverlapsPrefix` |
| `ip_add` | `(ip string, n number)` | `string` | custom (overflow-safe arithmetic) |
| `ip_in_cidr` | `(ip string, cidr string)` | `bool` | `net/netip` `Prefix.Contains` |
| `ip_is_private` | `(ip string)` | `bool` | custom (RFC 1918/4193/6598 table) |
| `ip_subtract` | `(a string, b string)` | `number` | custom (overflow-safe arithmetic) |
| `ip_to_mixed_notation` | `(ip string)` | `string` | custom (RFC 5952 mixed format) |
| `ip_version` | `(ip string)` | `number` | `net/netip` `Addr.Is4` |
| `ipv4_to_ipv4_mapped` | `(ipv4 string)` | `string` | custom (RFC 4291 §2.5.5.2) |
| `nat64_extract` | `(ipv6 string [, nat64_prefix string])` | `string` | custom (RFC 6052 §2.2) |
| `nat64_prefix_valid` | `(prefix string)` | `bool` | custom (RFC 6052 rules) |
| `nat64_synthesize` | `(ipv4 string, prefix string [, use_hex bool])` | `string` | custom (RFC 6052 §2.2) |
| `nat64_synthesize_cidr` | `(ipv4_cidr string, prefix string [, use_hex bool])` | `string` | custom (RFC 6052 §2.2) |
| `nat64_synthesize_cidrs` | `(ipv4_cidrs list(string), prefix string [, use_hex bool])` | `list(string)` | custom (RFC 6052 §2.2) |
| `nptv6_translate` | `(ipv6 string, from_prefix string, to_prefix string)` | `string` | custom (RFC 6296 checksum-neutral) |
| `range_to_cidrs` | `(first_ip string, last_ip string)` | `list(string)` | netipx `IPRange.Prefixes()` |

### RFC 1149 / RFC 2549 — IP over Avian Carriers

[RFC 1149](https://www.rfc-editor.org/rfc/rfc1149) ("A Standard for the Transmission of IP Datagrams on Avian Carriers", April 1990) and [RFC 2549](https://www.rfc-editor.org/rfc/rfc2549) ("IP over Avian Carriers with Quality of Service", April 1999) are two of the IETF's most well-known April Fools' RFCs. They specify, with completely straight faces, the frame format, MTU, and QoS framework for transmitting IP datagrams via homing pigeon. The implementation here is faithful to the metrics the RFCs imply — chosen constants, citation-mapped output fields, and the verbatim §3 frame-format string — in the same spirit as `pi_digit` is faithful to RFC 3091 over in [Numerics](#numerics-functions).

| Function | Signature | Returns | Backed by |
|---|---|---|---|
| `pigeon_throughput` | `(distance_km number, payload_bytes number, altitude_m number)` | `object` | custom (RFC 1149 §3 + RFC 2549 §§3, 6). Output object: `mtu_bytes`, `birds_required`, `per_bird_payload_bytes`, `cruise_speed_kmh`, `flight_time_seconds`, `throughput_bps`, `packet_loss_probability`, `effective_throughput_bps`, `qos_class`, `frame_format`, `rfc_citations`. |

Per-function documentation lives under [`docs/functions/`](docs/functions/) and on [registry.terraform.io](https://registry.terraform.io/providers/keeleysam/burnham/latest/docs).

## Query and Patch Functions

Pure functions for querying and patching decoded structures, so that manifest overlays and field extractions can live in plain HCL instead of regex hacks or `templatefile()`-driven preprocessors. Every function is deterministic and evaluates at plan time.

| Function | Signature | Returns | Backed by |
|---|---|---|---|
| `jmespath_query` | `(value dynamic, expression string)` | `dynamic` | [jmespath-community/go-jmespath](https://github.com/jmespath-community/go-jmespath) |
| `json_merge_patch` | `(value dynamic, patch dynamic)` | `dynamic` | [evanphx/json-patch](https://github.com/evanphx/json-patch), [RFC 7396](https://www.rfc-editor.org/rfc/rfc7396) |
| `json_patch` | `(value dynamic, patch list(object))` | `dynamic` | [evanphx/json-patch](https://github.com/evanphx/json-patch), [RFC 6902](https://www.rfc-editor.org/rfc/rfc6902) |
| `jsonpath_query` | `(value dynamic, expression string)` | `dynamic` (list) | [theory/jsonpath](https://github.com/theory/jsonpath), conformant with [RFC 9535](https://www.rfc-editor.org/rfc/rfc9535.html) |

Per-function documentation lives under [`docs/functions/`](docs/functions/) and on [registry.terraform.io](https://registry.terraform.io/providers/keeleysam/burnham/latest/docs).

## Numerics Functions

Pure mathematical / standards-based functions that don't fit the other families. All deterministic, all evaluating at plan time. Two clusters today: an exhaustively faithful implementation of [RFC 3091](https://www.rfc-editor.org/rfc/rfc3091) — the *Pi Digit Generation Protocol* — and a small set of statistics and math helpers that fill gaps in Terraform's built-ins.

### RFC 3091 — Pi Digit Generation Protocol

| Function | Signature | Returns | Backed by |
|---|---|---|---|
| `pi_approximate_digit` | `(n number)` | `string` `"<n>:<digit>"` | RFC 3091 §2.2 UDP reply for 22/7. Unbounded `n`. |
| `pi_approximate_digits` | `(count number)` | `string` (count chars) | RFC 3091 §1.1 TCP service for 22/7. Period-6 cycle `"142857"`. |
| `pi_digit` | `(n number)` | `string` `"<n>:<digit>"` | RFC 3091 §2.1.2 UDP reply for π. Embedded table of the first ⌊π × 10⁶⌋ = 3,141,592 digits, IEEE 754-2008 DPD-packed. |
| `pi_digits` | `(count number)` | `string` (count chars) | RFC 3091 §1 TCP service for π. Same packed table. |

### Statistics

Operate on `list(number)`. Empty input is always an error — a statistic of zero observations is undefined. Variance and standard deviation use the **population** formulas (divide by N, matching numpy's default); for sample statistics multiply variance by `N / (N − 1)` explicitly. Percentile uses the linear-interpolation method (Hyndman & Fan Type 7 — the default in numpy, R, and Excel `PERCENTILE.INC`).

| Function | Signature | Returns | Backed by |
|---|---|---|---|
| `mean` | `(numbers list(number))` | `number` | `math/big` (arbitrary precision) |
| `median` | `(numbers list(number))` | `number` | sort + `math/big` |
| `mode` | `(numbers list(number))` | `list(number)` | sort + bucket; multimodal-safe |
| `percentile` | `(numbers list(number), p number)` | `number` | linear interpolation, Type 7 |
| `stddev` | `(numbers list(number))` | `number` | `math/big.Float.Sqrt` of population variance |
| `variance` | `(numbers list(number))` | `number` | population variance, `math/big` |

### Math helpers

| Function | Signature | Returns | Backed by |
|---|---|---|---|
| `clamp` | `(value number, min_val number, max_val number)` | `number` | comparison; errors when `min_val > max_val` |
| `mod_floor` | `(a number, b number)` | `number` | floor-modulo: `a − b·⌊a/b⌋`. Sign of divisor, not dividend (unlike Terraform's built-in `%`). |

Per-function documentation lives under [`docs/functions/`](docs/functions/) and on [registry.terraform.io](https://registry.terraform.io/providers/keeleysam/burnham/latest/docs).

## Identifiers Functions

Pure functions that produce stable, plan-time identifiers. **Determinism is the point**: same inputs always produce the same output, so a Terraform plan that references these functions does not churn on re-apply. Useful for naming resources, deriving database keys from logical names, and generating sortable IDs without leaning on a random provider that would force every plan to reseed.

| Function | Signature | Returns | Backed by |
|---|---|---|---|
| `nanoid` | `(seed string [, options object])` | `string` | HMAC-SHA-256 in counter mode, deterministic from `seed`. Default 21-char URL-safe alphabet. |
| `petname` | `(seed string [, options object])` | `string` | HMAC-SHA-256(`seed`) → indices into embedded 64-entry adjective / adverb / noun lists. Heroku-style names. |
| `uuid_inspect` | `(uuid string)` | `object` | [google/uuid](https://github.com/google/uuid). Returns `{version, variant, timestamp, unix_ts_ms}`. |
| `uuid_v5` | `(namespace string, name string)` | `string` | [google/uuid](https://github.com/google/uuid) `NewSHA1` per [RFC 9562 §5.5](https://www.rfc-editor.org/rfc/rfc9562#name-uuid-version-5). Namespace accepts `"dns"` / `"url"` / `"oid"` / `"x500"` shorthands or any UUID. |
| `uuid_v7` | `(timestamp string, entropy string)` | `string` | Custom [RFC 9562 §5.7](https://www.rfc-editor.org/rfc/rfc9562#name-uuid-version-7) implementation. 48-bit Unix-ms timestamp + 74 HMAC-derived bits keyed by `entropy`, so `(timestamp, entropy)` is deterministic. |

Per-function documentation lives under [`docs/functions/`](docs/functions/) and on [registry.terraform.io](https://registry.terraform.io/providers/keeleysam/burnham/latest/docs).

## Text Functions

Pure functions for string manipulation and small text-rendering tasks. Carefully scoped to *not* duplicate Terraform core (which has `lower`, `upper`, `replace`, etc.) or [`northwood-labs/corefunc`](https://registry.terraform.io/providers/northwood-labs/corefunc/latest) (which owns case-conversion). Text-rendering is included here too — `cowsay` and `qr_ascii` produce text artefacts and feel at home next to text manipulation.

| Function | Signature | Returns | Backed by |
|---|---|---|---|
| `cowsay` | `(message string [, options object])` | `string` | self-contained; no external `cowsay` binary involved |
| `levenshtein` | `(a string, b string)` | `number` | classic two-row DP, codepoint-aware |
| `qr_ascii` | `(payload string [, options object])` | `string` | [`rsc.io/qr`](https://pkg.go.dev/rsc.io/qr) + half-block Unicode rendering |
| `slugify` | `(s string [, options object])` | `string` | [`gosimple/slug`](https://github.com/gosimple/slug) — Unicode → ASCII transliteration |
| `unicode_normalize` | `(s string, form string)` | `string` | [`golang.org/x/text/unicode/norm`](https://pkg.go.dev/golang.org/x/text/unicode/norm); UAX #15 |
| `wrap` | `(s string, width number)` | `string` | [`mitchellh/go-wordwrap`](https://github.com/mitchellh/go-wordwrap) |

Per-function documentation lives under [`docs/functions/`](docs/functions/) and on [registry.terraform.io](https://registry.terraform.io/providers/keeleysam/burnham/latest/docs).

## Cryptography Functions

Pure functions for keyed hashing, key derivation, and certificate / CSR / ASN.1 inspection. The headline wins are `x509_inspect` and `csr_inspect`: cert metadata becomes first-class HCL instead of a thing you regex out of a `tls_*.crt`. `hmac` and `hkdf` close another long-standing gap — webhook signing and per-tenant key derivation no longer need an `external` data source.

| Function | Signature | Returns | Backed by |
|---|---|---|---|
| `asn1_decode` | `(der_base64 string)` | recursive object | `encoding/asn1.RawValue` walked by hand |
| `csr_inspect` | `(pem string)` | object | stdlib `crypto/x509` (`ParseCertificateRequest`) |
| `hkdf` | `(algorithm string, secret string, salt string, info string, length number)` | `string` (hex) | [`golang.org/x/crypto/hkdf`](https://pkg.go.dev/golang.org/x/crypto/hkdf), RFC 5869 |
| `hmac` | `(algorithm string, key string, message string)` | `string` (hex) | stdlib `crypto/hmac`, RFC 2104 |
| `pem_decode` | `(pem string)` | `list(object)` | stdlib `encoding/pem`, RFC 7468 |
| `x509_fingerprint` | `(pem string, algorithm string)` | `string` (hex) | stdlib SHA-1/SHA-2 over the cert's DER bytes |
| `x509_inspect` | `(pem string)` | object | stdlib `crypto/x509` (`ParseCertificate`) |

`hmac` and `hkdf` accept inputs as raw bytes (the framework hands the function a UTF-8 string verbatim). HCL string literals only support `\uNNNN` escape sequences for non-ASCII bytes, and those are emitted as their UTF-8 encoding rather than as raw byte values — so RFC test vectors that exercise high-byte inputs aren't directly representable in HCL. ASCII-only inputs round-trip cleanly; for arbitrary-byte inputs, base64-encode and `base64decode(...)` first.

Per-function documentation lives under [`docs/functions/`](docs/functions/) and on [registry.terraform.io](https://registry.terraform.io/providers/keeleysam/burnham/latest/docs).

## Geographic Functions

Pure functions for geocoding — turning `(latitude, longitude)` pairs into short alphanumeric strings and back. Geohash and Open Location Code (Plus codes) are the two formats actually used in production: the former for spatial indexing in datastores, the latter for human-shareable location strings.

| Function | Signature | Returns | Backed by |
|---|---|---|---|
| `geohash_decode` | `(hash string)` | `object` | [`mmcloughlin/geohash`](https://pkg.go.dev/github.com/mmcloughlin/geohash) `Decode`+`BoundingBox` |
| `geohash_encode` | `(latitude number, longitude number, precision number)` | `string` | [`mmcloughlin/geohash`](https://pkg.go.dev/github.com/mmcloughlin/geohash) `EncodeWithPrecision` |
| `pluscode_decode` | `(code string)` | `object` | [Google `open-location-code`](https://github.com/google/open-location-code/tree/main/go) `CheckFull`+`Decode` |
| `pluscode_encode` | `(latitude number, longitude number, length number)` | `string` | Google `open-location-code` `Encode` |

Both decoders return `{latitude, longitude, lat_min, lat_max, lon_min, lon_max, …}` so callers can use either the centre or the cell extents. Plus code decoder additionally returns the code's `length`.

Per-function documentation lives under [`docs/functions/`](docs/functions/) and on [registry.terraform.io](https://registry.terraform.io/providers/keeleysam/burnham/latest/docs).

## Installation

```hcl
terraform {
  required_providers {
    burnham = {
      source = "keeleysam/burnham"
    }
  }
}
```

No provider configuration is needed — Burnham is a pure function provider with no resources, data sources, or remote API calls.

## Examples

A short tour. See [`examples/functions/`](examples/functions/) for a working snippet per function and [`docs/functions/`](docs/functions/) for the rendered per-function reference.

### Pretty-printed JSON

Terraform's built-in `jsonencode` produces a single line. Burnham's gives you human-editable output and configurable indentation.

```hcl
locals {
  policy = provider::burnham::jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["s3:GetObject", "s3:ListBucket"]
      Resource = ["arn:aws:s3:::my-bucket/*"]
    }]
  })

  # Two-space indent:
  policy_spaces = provider::burnham::jsonencode(local.policy_input, { indent = "  " })
}
```

### Apple plist from scratch

Build a configuration profile or `.mobileconfig` payload as a native HCL value. Output is XML by default; pass `format = "binary"` for a base64-encoded binary plist or `format = "openstep"` for OpenStep/GNUStep.

```hcl
output "wifi_profile" {
  value = provider::burnham::plistencode({
    PayloadDisplayName       = "WiFi - Corporate"
    PayloadIdentifier        = "com.example.wifi"
    PayloadType              = "Configuration"
    PayloadVersion           = 1
    PayloadRemovalDisallowed = true
    PayloadContent = [{
      PayloadType    = "com.apple.wifi.managed"
      AutoJoin       = true
      SSID_STR       = "CorpNet"
      EncryptionType = "WPA2"
    }]
  })
}
```

### Dual-stack CIDR merge

`cidr_merge` accepts a mixed IPv4/IPv6 list and returns merged ranges in both families — useful when you've collected blocks from multiple sources and want a single canonical, non-overlapping list to feed into a security group, route table, or firewall allowlist.

```hcl
locals {
  ipv4_blocks = ["10.0.0.0/24", "10.0.1.0/24", "10.0.2.0/23"]
  ipv6_blocks = ["2001:db8::/65", "2001:db8::8000:0:0:0/65", "2001:db9::/64"]

  // Single call, both families collapse independently:
  merged = provider::burnham::cidr_merge(concat(local.ipv4_blocks, local.ipv6_blocks))
  // → ["10.0.0.0/22", "2001:db8::/64", "2001:db9::/64"]
}
```

The dual-stack-allowlist pattern combines `cidr_merge` with `nat64_synthesize_cidrs` to extend an existing IPv4 allowlist into IPv6 space so IPv6-only clients reaching the service through a NAT64 gateway end up matching the same rule:

```hcl
locals {
  ipv4_allow = ["203.0.113.0/24", "198.51.100.0/24"]
  ipv6_allow = provider::burnham::nat64_synthesize_cidrs(local.ipv4_allow, "64:ff9b::/96")
  full_allow = provider::burnham::cidr_merge(concat(local.ipv4_allow, local.ipv6_allow))
}
```

## Requirements

- Terraform >= 1.8 (provider-defined functions)

## Developing

### Building

```sh
go build ./...
```

### Testing

The test suite has two layers:

**Unit tests** test internal Go functions directly — the type conversion engine, tagged object handling, edge cases, and error paths. They're fast and don't require Terraform.

**Acceptance tests** (`TestAcc_*`) run each provider function through the real Terraform plugin protocol using `terraform-plugin-testing`. They validate that functions work end-to-end as Terraform would call them — argument parsing, type coercion, dynamic returns, and error reporting. These require a `terraform` binary on your PATH (>= 1.8).

```sh
# Run everything (unit + acceptance)
make test

# Run only unit tests
go test ./internal/provider/ -run '^Test[^A]' -count=1 -v

# Run only acceptance tests
go test ./internal/provider/ -run '^TestAcc_' -count=1 -v

# Coverage
make cover
```

### Local testing with Terraform

Build the provider and create a dev override so Terraform uses your local binary:

```sh
go build -o terraform-provider-burnham .
```

Add to `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "keeleysam/burnham" = "/path/to/terraform-burnham"
  }
  direct {}
}
```

Then either run `terraform console` from anywhere (no config required — type `provider::burnham::cidr_merge(...)` interactively), or `terraform plan` against any of the per-function modules in `examples/functions/<name>/`. No `terraform init` needed with dev overrides.

## License

MPL-2.0
