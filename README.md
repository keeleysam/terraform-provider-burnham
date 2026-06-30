# Terraform Provider Burnham

<p align="center">
  <img src="assets/logo.svg" alt="Burnham" width="300" height="300">
</p>

> *"Aim high in hope and work, remembering that a noble, logical diagram once recorded will never die."*
> — Daniel Burnham

In 1909, Daniel Burnham published the [Plan of Chicago](https://en.wikipedia.org/wiki/Burnham_Plan_of_Chicago) — a comprehensive blueprint that transformed a sprawling, chaotic city into something coherent and enduring. He believed that good planning wasn't just about what you build, but about making the plan itself clear, readable, and maintainable for generations to come.

Terraform plans deserve the same treatment. But today, when your Terraform needs to work with structured data formats like property lists or human-edited JSON, do real arithmetic on IP address space, or apply environment overlays to a base manifest, you're stuck with workarounds. Shelling out to external tools, embedding raw strings, pasting opaque expressions that obscure what the plan is actually doing.

Burnham fixes this. It's a pure function provider — no resources, no data sources, no API calls — that fills the operations Terraform's expression language can't handle cleanly on its own. Structured data formats and network arithmetic at the foundation; query and patch over decoded values; deterministic identifiers, text manipulation, certificate inspection, and geographic encoding alongside; and a small numerics library covering RFC 3091 and a handful of statistics helpers.

Your configuration profiles, ACL policies, and structured documents become first-class citizens in your Terraform plans, not opaque blobs passed through `file()` and hoped for the best. Your network plans show set arithmetic on CIDRs in plain HCL instead of `templatefile()`-driven Python preprocessors. Your manifest overlays apply RFC 7396 merge patches in one expression rather than a chain of `merge()` and `try()` calls. Your TLS certificates surface their expiry, SANs, and fingerprints as structured fields instead of opaque base64 blobs.

The result is Terraform code that reads like a blueprint — clear, logical, and built to last.

Burnham is organized into eleven families of functions:

- **[Expression Language Functions](#expression-language-functions)**: build, validate, format, decode, and evaluate expression-language strings from HCL data. [CEL](https://cel.dev) (Common Expression Language) for GCP IAM / Access Context Manager, Kubernetes, and any other CEL sink; [Okta Expression Language](https://developer.okta.com/docs/reference/okta-expression-language/) for Okta group rules, profile mappings, and policy conditions.
- **[Structured Data Functions](#structured-data-functions)** — encode/decode for JSON (pretty), HuJSON, plist, INI, CSV, YAML, .reg, VDF, KDL, NDJSON, MessagePack, CBOR, dotenv, Java .properties, Apple .strings, and general HCL.
- **[Compression Functions](#compression-functions)** — `base64zopfli` (RFC 1952 gzip via Zopfli, a tighter drop-in for `base64gzip`) and `base64brotli` (RFC 7932 Brotli).
- **[Encoding Functions](#encoding-functions)** — byte codecs that fill core gaps: hex (`hexencode`/`hexdecode`), base64 and base32 with alphabet/padding options and lenient decoders, and `urlencode` (with `query`/`path`/`component` modes) / `urldecode` (the decoder core lacks).
- **[Networking Functions](#networking-functions)** — CIDR set operations, queries, IP arithmetic, NAT64 (RFC 6052), NPTv6 (RFC 6296), IPAM helpers, and a faithful RFC 1149 / RFC 2549 (IP over Avian Carriers) throughput calculator.
- **[Query and Patch Functions](#query-and-patch-functions)** — jq, JMESPath, JSONPath (RFC 9535), JSON Patch (RFC 6902), and JSON Merge Patch (RFC 7396) over decoded structures.
- **[Numerics Functions](#numerics-functions)** — RFC 3091 (Pi Digit Generation Protocol), statistics, and small math helpers.
- **[Identifiers Functions](#identifiers-functions)** — deterministic UUIDs (v5, v7), Nano ID, and petname.
- **[Text Functions](#text-functions)** — Unicode normalization, transliterating slugify, Levenshtein distance, word-wrap, dedent, cowsay, ASCII QR.
- **[Cryptography Functions](#cryptography-functions)** — HMAC (RFC 2104), HKDF (RFC 5869), PEM block decoding, X.509 / CSR inspection and fingerprinting, generic ASN.1 BER/DER decoding, deterministic ECDSA P-256 + Ed25519 key derivation, deterministic X.509 self-signing (RFC 5280) and CMS/PKCS#7 signing (RFC 5652) — ECDSA via RFC 6979 deterministic `k`, Ed25519 via naturally-deterministic PureEdDSA (RFC 8032 / RFC 8419) — and RFC 1751 human-readable key encoding (`btoe` / `etob`).
- **[Geographic Functions](#geographic-functions)** — geohash and Open Location Code (Plus codes), encode and decode.

## Expression Language Functions

Build, check, format, round-trip, and (for CEL) evaluate expression-language strings from HCL data at plan time, with no string templating and no manual quote escaping. All are pure and deterministic; the `*encode` functions build the expression from a structured HCL value, so references, enums, and lists flow in from Terraform variables and loops.

### CEL

[CEL](https://cel.dev) (Common Expression Language) is the expression language behind GCP IAM and Access Context Manager conditions, Kubernetes admission and CRD validation policies, Envoy RBAC, and more. `celencode` builds the expression by mirroring the CEL canonical AST (`cel/expr/syntax.proto`).

| Function | Purpose |
|----------|---------|
| `celencode` | Build a CEL string from an HCL data tree (a readable surface notation or the canonical `syntax.proto` field-name notation, mixable). References are marked `{ ident = "..." }`; everything else is a literal. |
| `celvalidate` | Report whether a string is syntactically valid CEL (returns a bool, does not fail the plan). `{ strict = true }` checks against base CEL with no extensions. |
| `celformat` | Parse and return the canonical, optionally pretty-printed CEL string (fails on invalid input). |
| `celdecode` | The inverse of `celencode`: parse a CEL string back into the data tree, in a chosen notation. `celencode(celdecode(x))` round-trips to the canonical form of `x`. |
| `celevaluate` | Evaluate a *standard* CEL expression against variable bindings and return the result. Runs cel-go's standard library plus extensions; host-specific functions (GCP `inIpRange`, Kubernetes `quantity`, and the like) are not implemented. |

The encode / validate / format / decode functions are syntax-only and dialect-neutral, so they accept any function name and suit any CEL sink. Only `celevaluate` actually runs the expression, so it is limited to standard CEL. Backed by [cel-go](https://github.com/google/cel-go) (and [celfmt](https://github.com/elastic/celfmt) for pretty-printing).

### Okta Expression Language

[Okta Expression Language](https://developer.okta.com/docs/reference/okta-expression-language/) (OEL) is a subset of Spring Expression Language used by Okta group rules, profile mappings, sign-on policy conditions, and Okta Identity Governance policies. `oelencode` assembles these expressions from HCL data so they are no longer hand-written, quote-escaped strings.

| Function | Purpose |
|----------|---------|
| `oelencode` | Build an OEL string from an HCL data tree. References are marked `{ ident = "user.department" }`; everything else is a literal. Operators are tokens or aliases (`==`/`eq`, `and`/`or`/`not`, `+`, `cond`, `elvis`, `matches`); calls take a `class`/`method` form (`String.stringContains(...)`), a bare `function` form (`isMemberOfAnyGroup(...)`, `substringBefore(...)`), or a receiver `target`/`method` form (`user.getInternalProperty("status")`, `user.isMemberOf({...})`); plus `select`, `index`, `project`, and `map`. |
| `oelvalidate` | Report whether a string is syntactically valid OEL (returns a bool, does not fail the plan). |
| `oelformat` | Parse and return the canonical OEL string (normalized spacing and quoting; fails on invalid input). |
| `oeldecode` | The inverse of `oelencode`: parse an OEL string back into the data tree, so `oelencode(oeldecode(x))` round-trips to the canonical form of `x`. |
| `oelevaluate` | Evaluate an OEL expression against a sample `user` profile and group memberships and return the result, for previewing or testing a group rule at plan time. A local approximation of the group-rule subset, not Okta's server-side engine. |

`oelencode` output is parsed back before it is returned, so it never emits a syntactically invalid expression, and it is byte-identical to `oelformat`'s canonical form. `oelencode`, `oelvalidate`, `oelformat`, and `oeldecode` cover the full documented OEL grammar (the classic namespaced subset plus receiver method calls, the Identity Engine method dialect, `isMemberOf({...})`, indexing, projection, Elvis, and `matches`); `oelevaluate` is limited to the group-rule subset it can actually evaluate. Backed by a fork of [okta-expression-parser](https://github.com/keeleysam/okta-expression-parser) that extends the parser to the full grammar (pending upstream contribution).

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
| JSON (pretty-printed) | `jsonencode` | — | Terraform has `jsondecode` built-in. Does not HTML-escape `<` `>` `&` by default (`escape_html` option to opt in); configurable `indent` |
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

## Compression Functions

Compress a string and base64-encode the result, for payloads like EC2 `user_data` that bump against size limits. Terraform's built-in `base64gzip` is the standards baseline; these are opt-in alternatives that trade a little plan-time CPU (or a consumer-side decompressor) for smaller output. Both are pure and **deterministic** — identical input and options always produce byte-identical output, so plans never churn.

| Function | Format | Decompresses with | Notes |
|--------|--------|--------|-------|
| `base64zopfli` | gzip ([RFC 1952](https://www.rfc-editor.org/rfc/rfc1952) / [RFC 1951](https://www.rfc-editor.org/rfc/rfc1951)) | `gunzip`, `zcat`, any gzip decoder | Drop-in replacement for `base64gzip` using [Zopfli](https://github.com/google/zopfli)'s iterative DEFLATE encoder — a few percent smaller, consumer side unchanged. Header pinned to `MTIME=0` / `XFL=2` / `OS=255`. Optional `{ iterations }` (default 15). |
| `base64brotli` | Brotli ([RFC 7932](https://www.rfc-editor.org/rfc/rfc7932)) | `brotli -d`, browser `Content-Encoding: br` | ~8–10% smaller than `base64gzip` on text, but requires a brotli decompressor on the consuming side. Optional `{ quality, lgwin }` (defaults 11 / 22). |

Both are pure Go (`CGO_ENABLED=0`), via [`foobaz/go-zopfli`](https://github.com/foobaz/go-zopfli) and [`andybalholm/brotli`](https://github.com/andybalholm/brotli). The RFC 7932 §10 encoder `mode` hint isn't exposed on `base64brotli` — the pure-Go encoder doesn't apply it (`text` and `generic` are byte-identical, `font` is unreachable), so it would be a no-op rather than an honest knob.

## Encoding Functions

Byte codecs that fill gaps in Terraform core. Core ships no hex decoder, no base32 codec, and no URL decoder, and its `base64encode` / `urlencode` lack options. Inputs are taken as raw bytes (the literal UTF-8 bytes of the string); decoders return a byte string, usually fed into another function rather than printed. The RFC 4648 codecs are pure and deterministic.

Family rule: **encoders take options to pick an output; decoders take an option only when the input is ambiguous** — `base64decode` needs none (its alphabets are disjoint), `base32decode` needs the alphabet (standard and hex overlap), `urldecode` needs the mode (`+` means space in a query, literal in a path). Where a core function of the same name exists, calling burnham's with no options matches core.

| Function | Signature | Notes |
|---|---|---|
| `hexencode` | `(input string)` | Bytes → lowercase hex |
| `hexdecode` | `(input string)` | Hex → bytes. Case-insensitive; ASCII whitespace ignored. Closes the gap that left `hmac` / `hkdf` unable to take a hex key directly |
| `base64encode` | `(input string [, options object])` | No options = standard padded (identical to core `base64encode`). `{ url_safe }` uses the §5 URL-safe alphabet; `{ padding = false }` omits `=` |
| `base64decode` | `(input string)` | Accepts either alphabet, padded or not, ASCII whitespace ignored — a superset of core `base64decode` |
| `base32encode` | `(input string [, options object])` | RFC 4648 base32 (core has none). No options = standard padded; `{ hex_alphabet }` uses the `0–9A–V` alphabet (NSEC3); `{ padding = false }` for TOTP-style secrets |
| `base32decode` | `(input string [, options object])` | Lenient: case-insensitive, padding optional, whitespace ignored. `{ hex_alphabet = true }` to decode the hex alphabet (can't be auto-detected — the alphabets overlap) |
| `urlencode` | `(input string [, options object])` | Percent-encode. No options = `query` mode (`application/x-www-form-urlencoded`, identical to core `urlencode`); `{ mode = "path" }` / `"component"` use RFC 3986 `%20` instead of `+` |
| `urldecode` | `(input string [, options object])` | Percent-decode — **the function core lacks entirely**. `{ mode }` controls `+`: space in `query` (default), literal in `path` |

Per-function documentation lives under [`docs/functions/`](docs/functions/) and on [registry.terraform.io](https://registry.terraform.io/providers/keeleysam/burnham/latest/docs).

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

[RFC 1149](https://www.rfc-editor.org/rfc/rfc1149) ("A Standard for the Transmission of IP Datagrams on Avian Carriers", 1990) and [RFC 2549](https://www.rfc-editor.org/rfc/rfc2549) ("IP over Avian Carriers with Quality of Service", 1999) specify the frame format, MTU, and QoS framework for transmitting IP datagrams via homing pigeon. The implementation is faithful to the metrics the RFCs imply — chosen constants, citation-mapped output fields, and the verbatim §3 frame-format string — in the same spirit as `pi_digit` is faithful to RFC 3091 over in [Numerics](#numerics-functions).

| Function | Signature | Returns | Backed by |
|---|---|---|---|
| `pigeon_throughput` | `(distance_km number, payload_bytes number, altitude_m number)` | `object` | custom (RFC 1149 §3 + RFC 2549 §§3, 6). Output object: `mtu_bytes`, `birds_required`, `per_bird_payload_bytes`, `cruise_speed_kmh`, `flight_time_seconds`, `throughput_bps`, `packet_loss_probability`, `effective_throughput_bps`, `qos_class`, `frame_format`, `rfc_citations`. |

### RFC 8771 — I-DUNNO

[RFC 8771](https://www.rfc-editor.org/rfc/rfc8771) ("The Internationalized Deliberately Unreadable Network Notation (I-DUNNO)", 2020) packs an IP address's bits into Unicode codepoints whose UTF-8 byte lengths carry the bits per §3 Table 1 (1/2/3/4-byte = 7/11/16/21 bits), with §4 mandating at least one multi-octet sequence and one IDNA2008-DISALLOWED character. The §5 worked example — `198.51.100.164` → U+0063, U+000C, U+006C, U+04A4 — round-trips through this encoder exactly. Decoder is the spec's intentionally-omitted §3.2 ("the machines will know how to do it").

| Function | Signature | Returns | Backed by |
|---|---|---|---|
| `ip_idunno_encode` | `(ip string)` | `string` | custom (RFC 8771 §§3–4) |
| `ip_idunno_decode` | `(encoded string)` | `string` | custom (RFC 8771 §3.2, reversed) |

Per-function documentation lives under [`docs/functions/`](docs/functions/) and on [registry.terraform.io](https://registry.terraform.io/providers/keeleysam/burnham/latest/docs).

## Query and Patch Functions

Pure functions for querying and patching decoded structures, so that manifest overlays and field extractions can live in plain HCL instead of regex hacks or `templatefile()`-driven preprocessors. Every function is deterministic and evaluates at plan time.

| Function | Signature | Returns | Backed by |
|---|---|---|---|
| `jmespath_query` | `(value dynamic, expression string)` | `dynamic` | [jmespath-community/go-jmespath](https://github.com/jmespath-community/go-jmespath) |
| `jq` | `(value dynamic, program string [, options object])` | `dynamic` (list) | [itchyny/gojq](https://github.com/itchyny/gojq) — pure-Go jq. Stream → list; `vars` bindings. `now`/`localtime` work but are non-deterministic; `env`/`$ENV` are empty (no host env), `input`/`inputs` error |
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
| `dedent` | `(s string)` | `string` | `textwrap.dedent` — strips the common leading whitespace from every line (the inverse of core `indent`) |
| `levenshtein` | `(a string, b string)` | `number` | classic two-row DP, codepoint-aware |
| `qr_ascii` | `(payload string [, options object])` | `string` | [`rsc.io/qr`](https://pkg.go.dev/rsc.io/qr) + half-block Unicode rendering |
| `slugify` | `(s string [, options object])` | `string` | [`gosimple/slug`](https://github.com/gosimple/slug) — Unicode → ASCII transliteration |
| `unicode_normalize` | `(s string, form string)` | `string` | [`golang.org/x/text/unicode/norm`](https://pkg.go.dev/golang.org/x/text/unicode/norm); UAX #15 |
| `wrap` | `(s string, width number)` | `string` | [`mitchellh/go-wordwrap`](https://github.com/mitchellh/go-wordwrap) |

Per-function documentation lives under [`docs/functions/`](docs/functions/) and on [registry.terraform.io](https://registry.terraform.io/providers/keeleysam/burnham/latest/docs).

## Cryptography Functions

Pure functions for keyed hashing, key derivation, certificate / CSR / ASN.1 inspection, and the four primitives needed to *construct* certs and CMS/PKCS#7 signatures deterministically. The inspection wins are `x509_inspect` and `csr_inspect`: cert metadata becomes first-class HCL instead of a thing you regex out of a `tls_*.crt`. `hmac` and `hkdf` close the symmetric-crypto gap — webhook signing and per-tenant key derivation no longer need an `external` data source. The signing chain (`{ecdsa_p256,ed25519}_key_from_seed` + `x509_self_sign` + `pkcs7_sign`) closes the asymmetric one: derive a stable signing identity from any seed, build a self-signed cert, and emit a byte-stable CMS SignedData — all without random state, suitable for Terraform-driven workflows where plan output must match apply output.

| Function | Signature | Returns | Backed by |
|---|---|---|---|
| `asn1_decode` | `(der_base64 string)` | recursive object | `encoding/asn1.RawValue` walked by hand |
| `btoe` | `(hex string)` | `string` (words) | custom (RFC 1751 §, faithful port). Encodes a key (hex, length a multiple of 8 bytes) as RFC 1751 English words |
| `csr_inspect` | `(pem string)` | object | stdlib `crypto/x509` (`ParseCertificateRequest`) |
| `ecdsa_p256_key_from_seed` | `(seed string)` | `string` (PEM PKCS#8) | stdlib `crypto/ecdsa` + `golang.org/x/crypto/hkdf` |
| `ed25519_key_from_seed` | `(seed string)` | `string` (PEM PKCS#8) | stdlib `crypto/ed25519` + `golang.org/x/crypto/hkdf`, RFC 8032 |
| `etob` | `(words string)` | `string` (hex) | custom (RFC 1751 §, faithful port). Decodes RFC 1751 English words back to a key, verifying the embedded parity |
| `hkdf` | `(algorithm string, secret string, salt string, info string, length number)` | `string` (hex) | [`golang.org/x/crypto/hkdf`](https://pkg.go.dev/golang.org/x/crypto/hkdf), RFC 5869 |
| `hmac` | `(algorithm string, key string, message string)` | `string` (hex) | stdlib `crypto/hmac`, RFC 2104 |
| `pem_decode` | `(pem string)` | `list(object)` | stdlib `encoding/pem`, RFC 7468 |
| `pkcs7_sign` | `(data string, private_key_pem string, cert_pem string)` | `string` (base64 DER) | [`digitorus/pkcs7`](https://github.com/digitorus/pkcs7) `SignWithoutAttr` + [`nspcc-dev/rfc6979`](https://github.com/nspcc-dev/rfc6979), RFC 5652 + RFC 6979 (ECDSA) / RFC 8419 (Ed25519) |
| `x509_fingerprint` | `(pem string, algorithm string)` | `string` (hex) | stdlib SHA-1/SHA-2 over the cert's DER bytes |
| `x509_inspect` | `(pem string)` | object | stdlib `crypto/x509` (`ParseCertificate`) |
| `x509_self_sign` | `(private_key_pem string, common_name string, serial string, not_before string, not_after string)` | `string` (PEM) | stdlib `crypto/x509.CreateCertificate` + [`nspcc-dev/rfc6979`](https://github.com/nspcc-dev/rfc6979), RFC 5280 + RFC 6979 (ECDSA) / RFC 8410 (Ed25519) |

`hmac` and `hkdf` accept inputs as raw bytes (the framework hands the function a UTF-8 string verbatim). HCL string literals only support `\uNNNN` escape sequences for non-ASCII bytes, and those are emitted as their UTF-8 encoding rather than as raw byte values — so RFC test vectors that exercise high-byte inputs aren't directly representable in HCL. ASCII-only inputs round-trip cleanly; for arbitrary-byte inputs, base64-encode and `base64decode(...)` first.

`x509_self_sign` and `pkcs7_sign` both accept ECDSA P-256 and Ed25519 keys and dispatch the right signing algorithm on the key type. ECDSA P-256 signs deterministically via [RFC 6979](https://www.rfc-editor.org/rfc/rfc6979) `k` derivation; Ed25519 is naturally deterministic by spec ([RFC 8032 §5.1.6](https://www.rfc-editor.org/rfc/rfc8032#section-5.1.6)) and signs as PureEdDSA per [RFC 8419](https://www.rfc-editor.org/rfc/rfc8419) (no pre-hash, SignerInfo `digestAlgorithm` set to `id-sha512` per §3). `pkcs7_sign` produces the "no signed attributes" CMS shape; it deliberately is *not* a general-purpose CMS-with-signed-attrs builder — `signingTime` would otherwise reintroduce non-determinism. RSA is intentionally out of scope; add it if a use case appears.

**macOS configuration-profile signing** uses ECDSA P-256: Apple's installer rejects Ed25519-signed `.mobileconfig` files at the keychain-import layer as of macOS 26.5. For everything else (OpenSSL `cms`, container signing, internal tooling) Ed25519 is the better default — shorter keys, simpler signatures, deterministic by spec.

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

Terraform's built-in `jsonencode` produces a single line. Burnham's gives you human-editable output and configurable indentation, and it leaves `<`, `>` and `&` as literal characters rather than escaping them to `<` / `>` / `&` — pass `{ escape_html = true }` if you need the escaped form (e.g. embedding JSON in an HTML `<script>`).

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
