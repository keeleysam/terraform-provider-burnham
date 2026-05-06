# Terraform Provider Burnham

<p align="center">
  <img src="assets/logo.svg" alt="Burnham" width="300" height="300">
</p>

> *"Aim high in hope and work, remembering that a noble, logical diagram once recorded will never die."*
> — Daniel Burnham

In 1909, Daniel Burnham published the [Plan of Chicago](https://en.wikipedia.org/wiki/Burnham_Plan_of_Chicago) — a comprehensive blueprint that transformed a sprawling, chaotic city into something coherent and enduring. He believed that good planning wasn't just about what you build, but about making the plan itself clear, readable, and maintainable for generations to come.

Terraform plans deserve the same treatment. But today, when your Terraform needs to work with structured data formats like property lists or human-edited JSON, or do real arithmetic on IP address space — set operations on CIDRs, NAT64 synthesis, range conversion — you're stuck with workarounds. Shelling out to external tools, embedding raw strings, pasting opaque expressions that obscure what the plan is actually doing.

Burnham fixes this. It's a pure function provider — no resources, no data sources, no API calls — that gives Terraform native fluency with the structured data formats and the network primitives it can't handle cleanly on its own.

Your configuration profiles, ACL policies, and structured documents become first-class citizens in your Terraform plans, not opaque blobs passed through `file()` and hoped for the best. Your network plans show set arithmetic on CIDRs in plain HCL instead of `templatefile()`-driven Python preprocessors.

The result is Terraform code that reads like a blueprint — clear, logical, and built to last.

Burnham is organized into two families of functions:

- **[Structured Data Functions](#structured-data-functions)** — encode/decode for JSON (pretty), HuJSON, plist, INI, CSV, YAML, .reg, VDF, KDL.
- **[Networking Functions](#networking-functions)** — CIDR set operations, queries, IP arithmetic, NAT64 (RFC 6052), NPTv6 (RFC 6296), and IPAM helpers.

## Structured Data Functions

| Format | Encode | Decode | Notes |
|--------|--------|--------|-------|
| JSON (pretty-printed) | `jsonencode` | — | Terraform has `jsondecode` built-in |
| HuJSON / JWCC | `hujsonencode` | `hujsondecode` | JSON with comments and trailing commas |
| Apple Property List | `plistencode` | `plistdecode` | XML (with comments), binary, and OpenStep formats |
| INI | `iniencode` | `inidecode` | Standard `[section]` / `key = value` files |
| CSV | `csvencode` | — | Terraform has `csvdecode` built-in |
| YAML | `yamlencode` | — | Block style, literal scalars, comments. Terraform has `yamldecode` built-in |
| Windows .reg | `regencode` | `regdecode` | Registry Editor export format with typed values and comments |
| Valve VDF | `vdfencode` | `vdfdecode` | Steam/Source engine config format |
| KDL | `kdlencode` | `kdldecode` | Modern document language, v1 and v2 |
| TOML | — | — | Use [Tobotimus/toml](https://registry.terraform.io/providers/Tobotimus/toml) instead |

### `jsonencode`

Encode a value as pretty-printed JSON. Unlike Terraform's built-in `jsonencode`, this produces human-readable output with configurable indentation.

```
provider::burnham::jsonencode(value, options?) → string
```

| Parameter | Type | Required | Description |
|---|---|---|---|
| `value` | `dynamic` | Yes | Any Terraform value to encode as JSON. |
| `options` | `object` | No | Options object. Supported keys: `indent` (string, default `"\t"`). |

**Returns:** A pretty-printed JSON `string`. Keys are sorted alphabetically. Whole numbers render without a decimal point (e.g. `1` not `1.0`).

---

### `hujsondecode`

Parse a [HuJSON](https://github.com/tailscale/hujson) string into a Terraform value. HuJSON (also known as [JWCC](https://nigeltao.github.io/blog/2021/json-with-commas-comments.html) — JSON With Commas and Comments) extends standard JSON with C-style comments (`//` and `/* */`) and trailing commas. It's a superset of JSONC (which only adds comments, not trailing commas) and is used by Tailscale ACL policies among others. Comments are stripped during decoding.

```
provider::burnham::hujsondecode(input) → dynamic
```

| Parameter | Type | Required | Description |
|---|---|---|---|
| `input` | `string` | Yes | A HuJSON string to parse. Standard JSON is also accepted. |

**Returns:** A `dynamic` value — the decoded structure. Objects become Terraform objects, arrays become tuples, strings/numbers/bools map directly. JSON numbers preserve precision.

---

### `hujsonencode`

Encode a Terraform value as a HuJSON string with trailing commas and pretty-printed formatting. Optionally add comments using a mirrored comment structure.

```
provider::burnham::hujsonencode(value, options?) → string
```

| Parameter | Type | Required | Description |
|---|---|---|---|
| `value` | `dynamic` | Yes | Any Terraform value to encode. |
| `options` | `object` | No | Options object (see below). |

**Options:**

| Key | Type | Default | Description |
|---|---|---|---|
| `indent` | `string` | `"\t"` | Indentation string for each level. |
| `comments` | `object` | none | A mirrored structure where string values become comments placed before the matching key. |

**Comments** mirror the shape of the data. Each key in the comments object corresponds to a key in the data. String values become comments — single-line strings produce `//` comments, multi-line strings (containing `\n`) produce `/* */` block comments. Nested objects in the comment map add comments to nested keys. Array elements are addressed by index as string keys (`"0"`, `"1"`, etc.). Comments for keys that don't exist in the data are silently ignored.

```hcl
provider::burnham::hujsonencode(
  { acls = [...], groups = {...} },
  {
    comments = {
      acls   = "Network ACL rules"
      groups = "Group membership"
    }
  }
)
# {
# 	// Network ACL rules
# 	"acls": [...],
# 	// Group membership
# 	"groups": {...},
# }
```

**Returns:** A HuJSON `string`. Multi-line objects and arrays get trailing commas. Small composites that fit on one line stay compact (standard hujson formatting behavior). Keys are sorted alphabetically.

---

### `plistdecode`

Parse an Apple property list into a Terraform value. Auto-detects XML, binary, OpenStep, and GNUStep formats. Also auto-detects base64-encoded input (for binary plists read with `filebase64()`).

```
provider::burnham::plistdecode(input) → dynamic
```

| Parameter | Type | Required | Description |
|---|---|---|---|
| `input` | `string` | Yes | A plist string from `file()`, or a base64-encoded plist from `filebase64()`. |

**Returns:** A `dynamic` value with this type mapping:

| Plist type | Terraform type | Notes |
|---|---|---|
| `<string>` | `string` | |
| `<integer>` | `number` | |
| `<real>` | `number` or `object` | Fractional (e.g. `3.14`) → plain number. Whole-number (e.g. `2.0`) → tagged: `{ __plist_type = "real", value = "2" }` to distinguish from `<integer>` |
| `<true/>` / `<false/>` | `bool` | |
| `<array>` | `tuple` | Heterogeneous element types supported |
| `<dict>` | `object` | Heterogeneous value types supported |
| `<date>` | `object` | Tagged: `{ __plist_type = "date", value = "2025-06-01T00:00:00Z" }` |
| `<data>` | `object` | Tagged: `{ __plist_type = "data", value = "base64..." }` |

Tagged objects for `<date>` and `<data>` use the same format as `plistdate()` and `plistdata()`, so decode-then-encode round-trips preserve types automatically.

---

### `plistencode`

Encode a Terraform value as an Apple property list.

```
provider::burnham::plistencode(value, options?) → string
```

| Parameter | Type | Required | Description |
|---|---|---|---|
| `value` | `dynamic` | Yes | The value to encode. Tagged objects from `plistdate()` and `plistdata()` are converted to native `<date>` and `<data>` elements. |
| `options` | `object` | No | Options object. Supported keys: `format` (string) — `"xml"` (default), `"binary"`, or `"openstep"`. `comments` (object) — mirrored structure where string values become `<!-- comment -->` in the XML output (XML format only). |

**Returns:** A plist `string`. When format is `"binary"`, the output is base64-encoded (since Terraform strings are UTF-8). Numbers with no fractional part become `<integer>`, otherwise `<real>`.

---

### `plistdate`

Create a tagged object representing a plist `<date>` value.

```
provider::burnham::plistdate(rfc3339) → dynamic
```

| Parameter | Type | Required | Description |
|---|---|---|---|
| `rfc3339` | `string` | Yes | An RFC 3339 timestamp, e.g. `"2025-06-01T00:00:00Z"`. Validated on input. |

**Returns:** A `dynamic` object: `{ __plist_type = "date", value = "2025-06-01T00:00:00Z" }`. Pass this to `plistencode` to produce a `<date>` element. This is the same format that `plistdecode` returns for `<date>` elements.

---

### `plistdata`

Create a tagged object representing a plist `<data>` (binary) value.

```
provider::burnham::plistdata(base64) → dynamic
```

| Parameter | Type | Required | Description |
|---|---|---|---|
| `base64` | `string` | Yes | A base64-encoded string, e.g. from `filebase64()`. Validated on input. |

**Returns:** A `dynamic` object: `{ __plist_type = "data", value = "base64..." }`. Pass this to `plistencode` to produce a `<data>` element. This is the same format that `plistdecode` returns for `<data>` elements.

---

### `plistreal`

Create a tagged object representing a plist `<real>` (floating-point) value. This is only needed for whole numbers that must encode as `<real>` instead of `<integer>` — fractional numbers like `3.14` are automatically encoded as `<real>` without this helper.

```
provider::burnham::plistreal(value) → dynamic
```

| Parameter | Type | Required | Description |
|---|---|---|---|
| `value` | `number` | Yes | The numeric value for the `<real>` element. |

**Returns:** A `dynamic` object: `{ __plist_type = "real", value = "2" }`. Pass this to `plistencode` to produce a `<real>` element. When `plistdecode` encounters a whole-number `<real>` (e.g. `<real>2</real>`), it returns the same tagged format, so round-trips preserve the integer/real distinction.

---

### `inidecode`

Parse an INI file into a Terraform value. The result is a map of section names to maps of key-value string pairs.

```
provider::burnham::inidecode(input) → dynamic
```

| Parameter | Type | Required | Description |
|---|---|---|---|
| `input` | `string` | Yes | An INI string to parse. |

**Returns:** A `dynamic` object of `{ section_name = { key = "value" } }`. Keys outside any `[section]` header (global keys) are placed under the `""` key. All values are strings — INI has no native type system. Comments (`;` and `#`) are stripped.

---

### `iniencode`

Encode a Terraform object as an INI file.

```
provider::burnham::iniencode(value) → string
```

| Parameter | Type | Required | Description |
|---|---|---|---|
| `value` | `dynamic` | Yes | An object of `{ section_name = { key = value } }`. The `""` key renders as global keys before any section header. All values are converted to strings. |

**Returns:** An INI `string` with `[section]` headers and `key = value` pairs. Sections are sorted alphabetically, with global keys first.

---

### `csvencode`

Encode a list of objects as a CSV string. Each object becomes a row, and object keys become columns. Terraform has `csvdecode` built-in but no `csvencode` — this fills that gap.

```
provider::burnham::csvencode(rows, options?) → string
```

| Parameter | Type | Required | Description |
|---|---|---|---|
| `rows` | `dynamic` | Yes | A list of objects to encode as CSV rows. |
| `options` | `dynamic` | No | An options object (see below). Pass at most one. |

**Options object:**

| Key | Type | Default | Description |
|---|---|---|---|
| `columns` | `list(string)` | auto-detect (sorted) | Column names in the desired output order. Columns not in a row produce empty cells. |
| `no_header` | `bool` | `false` | If `true`, omit the header row. |

**Returns:** A CSV `string`. Values are converted to strings: numbers render as their string representation, bools as `"true"`/`"false"`, nulls as empty strings. Nested values (lists, objects) are not supported and will error.

**Note on types:** CSV has no type system. All values are flattened to strings during encoding. If you round-trip through `csvencode` → Terraform's `csvdecode`, numbers and bools will come back as strings (e.g. `42` → `"42"`, `true` → `"true"`). This is inherent to the CSV format.

---

### `yamlencode`

Encode a value as YAML with full formatting control. Unlike Terraform's built-in `yamlencode`, this defaults to block style, uses literal block scalars (`|`) for multi-line strings, and supports comments.

```
provider::burnham::yamlencode(value, options?) → string
```

| Parameter | Type | Required | Description |
|---|---|---|---|
| `value` | `dynamic` | Yes | The value to encode as YAML. |
| `options` | `object` | No | Options object (see below). |

**Options:**

| Key | Type | Default | Description |
|---|---|---|---|
| `indent` | `number` | `2` | Spaces per indentation level. |
| `flow_level` | `number` | `0` | Nesting depth at which to switch to flow style. `0` = all block, `-1` = all flow. |
| `multiline` | `string` | `"literal"` | Multi-line string style: `"literal"` (`\|`), `"folded"` (`>`), or `"quoted"`. |
| `quote_style` | `string` | `"auto"` | String quoting: `"auto"`, `"double"`, or `"single"`. |
| `null_value` | `string` | `"null"` | Null rendering: `"null"`, `"~"`, or `""`. |
| `sort_keys` | `bool` | `true` | Sort map keys alphabetically. |
| `dedupe` | `bool` | `false` | Deduplicate identical subtrees using YAML anchors (`&`) and aliases (`*`). |
| `comments` | `object` | none | Mirrored structure for `#` comments (same pattern as `hujsonencode`). |

**Returns:** A YAML `string` in block style by default. Multi-line strings use literal block scalars (`|`). Keys are sorted alphabetically unless `sort_keys = false`.

---

### `regdecode`

Parse a Windows Registry Editor export (.reg) file into a Terraform value. Auto-detects Version 4 (REGEDIT4) and Version 5 formats.

```
provider::burnham::regdecode(input) → dynamic
```

| Parameter | Type | Required | Description |
|---|---|---|---|
| `input` | `string` | Yes | A .reg file string to parse. |

**Returns:** A `dynamic` object of `{ "HKEY_...\\Path" = { "ValueName" = value } }`. REG_SZ values become plain strings. Other types use tagged objects:

| Registry type | Terraform representation |
|---|---|
| `REG_SZ` | plain string |
| `REG_DWORD` | `{ __reg_type = "dword", value = "42" }` |
| `REG_QWORD` | `{ __reg_type = "qword", value = "42" }` |
| `REG_BINARY` | `{ __reg_type = "binary", value = "48656c6c6f" }` (hex) |
| `REG_MULTI_SZ` | `{ __reg_type = "multi_sz", value = ["str1", "str2"] }` |
| `REG_EXPAND_SZ` | `{ __reg_type = "expand_sz", value = "%SystemRoot%\\system32" }` |
| `REG_NONE` | `{ __reg_type = "none", value = "hex..." }` |
| Default value (`@`) | key name is `"@"` |

---

### `regencode`

Encode a Terraform object as a Windows .reg file (Version 5).

```
provider::burnham::regencode(value, options?) → string
```

| Parameter | Type | Required | Description |
|---|---|---|---|
| `value` | `dynamic` | Yes | An object of `{ "HKEY_...\\Path" = { "ValueName" = value } }`. Plain strings become REG_SZ. Use helper functions for other types. |
| `options` | `object` | No | Options object. Supported keys: `comments` (object) — mirrored structure where string values become `; comment` lines above the matching key path or value name. |

**Returns:** A .reg file `string` with the `Windows Registry Editor Version 5.00` header.

**Helper functions** for typed registry values:

| Function | Creates | Example |
|---|---|---|
| `regdword(number)` | REG_DWORD | `regdword(42)` |
| `regqword(number)` | REG_QWORD | `regqword(1099511627776)` |
| `regbinary(hex_string)` | REG_BINARY | `regbinary("48656c6c6f")` |
| `regmulti(list)` | REG_MULTI_SZ | `regmulti(["path1", "path2"])` |
| `regexpandsz(string)` | REG_EXPAND_SZ | `regexpandsz("%SystemRoot%\\system32")` |

---

### `vdfdecode`

Parse a [Valve Data Format](https://developer.valvesoftware.com/wiki/KeyValues) (VDF) string into a Terraform value. VDF is the nested key-value format used by Steam and Source engine games.

```
provider::burnham::vdfdecode(input) → dynamic
```

| Parameter | Type | Required | Description |
|---|---|---|---|
| `input` | `string` | Yes | A VDF string to parse. |

**Returns:** A `dynamic` object. VDF only has strings and nested objects — all leaf values are strings. Comments (`//`) are stripped.

---

### `vdfencode`

Encode a Terraform object as a VDF string.

```
provider::burnham::vdfencode(value) → string
```

| Parameter | Type | Required | Description |
|---|---|---|---|
| `value` | `dynamic` | Yes | An object to encode. Values must be strings or nested objects. Numbers and bools are converted to strings. |

**Returns:** A VDF `string` with tab-indented Valve-style formatting.

---

### `kdldecode`

Parse a [KDL](https://kdl.dev/) document into a Terraform value. KDL is a modern document language where each node has a name, positional arguments, named properties, and children. Supports both KDL v1 and v2 input.

```
provider::burnham::kdldecode(input) → dynamic
```

| Parameter | Type | Required | Description |
|---|---|---|---|
| `input` | `string` | Yes | A KDL document string to parse. |

**Returns:** A `dynamic` list of node objects. Each node has: `name` (string), `args` (list of values), `props` (map of values), `children` (list of child nodes).

---

### `kdlencode`

Encode a list of node objects as a KDL document.

```
provider::burnham::kdlencode(value, options?) → string
```

| Parameter | Type | Required | Description |
|---|---|---|---|
| `value` | `dynamic` | Yes | A list of node objects with `name`, `args`, `props`, and `children` keys. |
| `options` | `object` | No | Options object. Supported keys: `version` (string) — `"v2"` (default) or `"v1"`. |

**Returns:** A KDL `string`. Default output is KDL v2 format.

## Networking Functions

Pure functions for IP/CIDR work that HCL alone can't express: set arithmetic on address space, normalizing mixed IPv4/IPv6 inputs, NAT64 / NPTv6 translation, and IPAM-style allocation. All functions are pure (no network calls, no state) and evaluate at plan time. Uses [`go4.org/netipx`](https://pkg.go.dev/go4.org/netipx) for set operations, prefix aggregation, and range conversion.

The **Backed by** column matters for understanding where bugs live. Functions backed by `netipx` or `net/netip` are thin parsing wrappers — if the logic is wrong, it's almost certainly in the upstream library, not here. Functions with custom or RFC-derived implementations are where this provider adds real logic of its own.

| Function | Signature | Returns | Backed by |
|---|---|---|---|
| `cidr_merge` | `(cidrs list(string))` | `list(string)` | netipx `IPSetBuilder` + `Prefixes()` |
| `cidr_subtract` | `(input list(string), exclude list(string))` | `list(string)` | netipx `IPSetBuilder` |
| `cidr_intersect` | `(a list(string), b list(string))` | `list(string)` | netipx `IPSetBuilder.Intersect` |
| `cidr_expand` | `(cidr string)` | `list(string)` | custom iterator |
| `cidr_enumerate` | `(cidr string, newbits number)` | `list(string)` | custom iterator |
| `range_to_cidrs` | `(first_ip string, last_ip string)` | `list(string)` | netipx `IPRange.Prefixes()` |
| `cidr_find_free` | `(pool list(string), used list(string), prefix_len number)` | `string\|null` | netipx `IPSet.RemoveFreePrefix()` |
| `ip_in_cidr` | `(ip string, cidr string)` | `bool` | `net/netip` `Prefix.Contains` |
| `cidrs_containing_ip` | `(ip string, cidrs list(string))` | `list(string)` | `net/netip` `Prefix.Contains` |
| `cidr_contains` | `(cidr string, other string)` | `bool` | netipx `IPSet.ContainsPrefix/Contains` |
| `cidr_overlaps` | `(a string, b string)` | `bool` | `net/netip` `Prefix.Overlaps` |
| `cidrs_overlap_any` | `(a list(string), b list(string))` | `bool` | netipx `IPSet.OverlapsPrefix` |
| `cidrs_are_disjoint` | `(cidrs list(string))` | `bool` | netipx `IPSet.OverlapsPrefix` |
| `cidr_host_count` | `(cidr string)` | `number` | custom (bit math) |
| `cidr_usable_host_count` | `(cidr string)` | `number` | custom (bit math + RFC 3021) |
| `cidr_first_ip` | `(cidr string)` | `string` | `net/netip` `Prefix.Addr` |
| `cidr_last_ip` | `(cidr string)` | `string` | netipx `RangeOfPrefix().To()` |
| `cidr_prefix_length` | `(cidr string)` | `number` | `net/netip` `Prefix.Bits` |
| `cidr_wildcard` | `(cidr string)` | `string` | custom (bit math, IPv4 only) |
| `ip_add` | `(ip string, n number)` | `string` | custom (overflow-safe arithmetic) |
| `ip_subtract` | `(a string, b string)` | `number` | custom (overflow-safe arithmetic) |
| `ip_version` | `(ip string)` | `number` | `net/netip` `Addr.Is4` |
| `cidr_version` | `(cidr string)` | `number` | `net/netip` `Prefix.Addr.Is4` |
| `ip_is_private` | `(ip string)` | `bool` | custom (RFC 1918/4193/6598 table) |
| `cidr_is_private` | `(cidr string)` | `bool` | custom (RFC 1918/4193/6598 table) |
| `cidr_filter_version` | `(cidrs list(string), version number)` | `list(string)` | custom |
| `nat64_prefix_valid` | `(prefix string)` | `bool` | custom (RFC 6052 rules) |
| `nat64_synthesize` | `(ipv4 string, prefix string [, use_hex bool])` | `string` | custom (RFC 6052 §2.2) |
| `nat64_extract` | `(ipv6 string [, nat64_prefix string])` | `string` | custom (RFC 6052 §2.2) |
| `nat64_synthesize_cidrs` | `(ipv4_cidrs list(string), prefix string [, use_hex bool])` | `list(string)` | custom (RFC 6052 §2.2) |
| `nat64_synthesize_cidr` | `(ipv4_cidr string, prefix string [, use_hex bool])` | `string` | custom (RFC 6052 §2.2) |
| `nptv6_translate` | `(ipv6 string, from_prefix string, to_prefix string)` | `string` | custom (RFC 6296 checksum-neutral) |
| `ip_to_mixed_notation` | `(ip string)` | `string` | custom (RFC 5952 mixed format) |
| `ipv4_to_ipv4_mapped` | `(ipv4 string)` | `string` | custom (RFC 4291 §2.5.5.2) |

### CIDR set operations

#### `cidr_merge`

```
cidr_merge(cidrs list(string)) → list(string)
```

Aggregates a list of CIDRs into the smallest equivalent set by removing redundant prefixes and combining sibling pairs into supernets.

```hcl
provider::burnham::cidr_merge(["10.0.0.0/24", "10.0.1.0/24"])
# → ["10.0.0.0/23"]

provider::burnham::cidr_merge(["10.0.0.0/24", "10.0.0.0/25"])
# → ["10.0.0.0/24"]  (the /25 is redundant)
```

**When to use:** reducing rule counts in security groups, route tables, or firewall allowlists before applying them. Cloud providers often cap total rules or charge per rule, so merging before applying avoids hitting limits.

---

#### `cidr_subtract`

```
cidr_subtract(input list(string), exclude list(string)) → list(string)
```

Returns the address space in `input` after removing all ranges covered by `exclude`. The result is automatically merged to the smallest equivalent set.

```hcl
provider::burnham::cidr_subtract(["10.0.0.0/8"], ["10.1.0.0/16"])
# → all of 10.0.0.0/8 except 10.1.0.0/16, expressed as 8 CIDRs
```

**When to use:** computing free address space — start with a large allocation, subtract ranges reserved for other teams or infrastructure, and the result is what you can allocate from.

---

#### `cidr_intersect`

```
cidr_intersect(a list(string), b list(string)) → list(string)
```

Returns the CIDRs representing address space present in both `a` and `b`.

```hcl
provider::burnham::cidr_intersect(
  ["10.0.0.0/8", "172.16.0.0/12"],
  ["10.100.0.0/16", "192.168.0.0/16"]
)
# → ["10.100.0.0/16"]
```

**When to use:** finding which of your on-premises ranges are reachable through a specific VPN tunnel or transit gateway; determining the subset of a vendor's advertised routes that overlaps with internal space.

---

#### `cidr_expand`

```
cidr_expand(cidr string) → list(string)
```

Returns every individual IP address in the CIDR as a list of strings. Returns an error if the CIDR contains more than 65536 addresses.

```hcl
provider::burnham::cidr_expand("10.0.0.0/30")
# → ["10.0.0.0", "10.0.0.1", "10.0.0.2", "10.0.0.3"]
```

**When to use:** when a resource expects a list of individual IPs rather than a CIDR — individual DNS A records, static DHCP reservations, or host-level firewall rules for a small management subnet.

---

#### `range_to_cidrs`

```
range_to_cidrs(first_ip string, last_ip string) → list(string)
```

Converts an inclusive IP range to the minimal list of CIDRs that exactly covers it. Both IPs must be the same address family.

```hcl
provider::burnham::range_to_cidrs("10.0.0.1", "10.0.0.6")
# → ["10.0.0.1/32", "10.0.0.2/31", "10.0.0.4/31", "10.0.0.6/32"]
```

**When to use:** cloud IP range feeds (AWS, GCP, Azure), ARIN/RIPE exports, and MaxMind GeoIP databases commonly publish ranges as first/last IP pairs. This converts them to CIDR notation for use in security groups or route tables.

---

#### `cidr_enumerate`

```
cidr_enumerate(cidr string, newbits number) → list(string)
```

Returns every possible subnet of size `(parent prefix length + newbits)` within `cidr`. Returns an error if the result would exceed 65536 subnets.

```hcl
provider::burnham::cidr_enumerate("10.0.0.0/24", 2)
# → ["10.0.0.0/26", "10.0.0.64/26", "10.0.0.128/26", "10.0.0.192/26"]
```

**When to use:** pre-computing all candidate subnets of a target size within a region block so you can later pick specific ones with `element()`. More predictable than `cidrsubnet()` when you need to see all options up front, or when feeding into a `for_each` that needs the full set.

---

### Query / validation

#### `ip_in_cidr`

```
ip_in_cidr(ip string, cidr string) → bool
```

Returns `true` if `ip` is contained within `cidr`.

```hcl
provider::burnham::ip_in_cidr("10.0.1.50", "10.0.1.0/24") # → true
provider::burnham::ip_in_cidr("10.0.2.50", "10.0.1.0/24") # → false
```

**When to use:** `variable` validation blocks — ensure a user-supplied IP (bastion host, NTP server, etc.) actually falls within the expected subnet, catching misconfigurations before any resources are created.

---

#### `cidrs_containing_ip`

```
cidrs_containing_ip(ip string, cidrs list(string)) → list(string)
```

Returns every CIDR in `cidrs` that contains `ip`. Returns an empty list if none match. Multiple CIDRs may match if the list contains overlapping prefixes.

```hcl
provider::burnham::cidrs_containing_ip("10.0.1.5", ["10.0.0.0/8", "10.0.1.0/24", "192.168.0.0/16"])
# → ["10.0.0.0/8", "10.0.1.0/24"]
```

**When to use:** routing decisions — given an observed IP, determine which VRF, VPC, or security zone it belongs to. Returns all matches so overlapping summary/specific routes are both visible.

---

#### `cidr_contains`

```
cidr_contains(cidr string, other string) → bool
```

Returns `true` if `cidr` fully contains `other`. `other` may be a bare IP address or a CIDR string.

```hcl
provider::burnham::cidr_contains("10.0.0.0/8", "10.1.2.0/24") # → true
provider::burnham::cidr_contains("10.0.0.0/8", "10.1.2.3")    # → true
provider::burnham::cidr_contains("10.0.0.0/8", "192.168.0.0/16") # → false
```

**When to use:** asserting parent/child subnet relationships — confirming that all spoke VPC CIDRs are subnets of the hub's supernet before configuring transit gateway route tables.

---

#### `cidr_overlaps`

```
cidr_overlaps(a string, b string) → bool
```

Returns `true` if CIDRs `a` and `b` share at least one address.

```hcl
provider::burnham::cidr_overlaps("10.0.0.0/24", "10.0.0.128/25") # → true
provider::burnham::cidr_overlaps("10.0.0.0/24", "10.0.1.0/24")   # → false
```

**When to use:** `variable` validation blocks — check that a new subnet CIDR doesn't conflict with an existing one before calling `aws_subnet` or `azurerm_subnet`. Overlapping subnets in the same VPC produce an opaque API error; catching it here gives a clear message instead.

---

#### `cidrs_overlap_any`

```
cidrs_overlap_any(a list(string), b list(string)) → bool
```

Returns `true` if any CIDR in `a` overlaps with any CIDR in `b`.

```hcl
provider::burnham::cidrs_overlap_any(
  ["10.4.0.0/16", "10.5.0.0/16"],           # proposed
  ["10.0.0.0/16", "10.1.0.0/16", "10.4.0.0/16"]  # existing
)
# → true
```

**When to use:** bulk conflict checks in `variable` validation — ensure a batch of new VPC CIDRs don't collide with any already-peered networks. Much cleaner than nested `for` loops in a condition expression.

---

#### `cidrs_are_disjoint`

```
cidrs_are_disjoint(cidrs list(string)) → bool
```

Returns `true` if no two CIDRs in the list overlap each other. Unlike `cidrs_overlap_any`, which compares two separate lists, this checks a single list against itself.

```hcl
provider::burnham::cidrs_are_disjoint(["10.0.0.0/24", "10.0.1.0/24"]) # → true
provider::burnham::cidrs_are_disjoint(["10.0.0.0/8",  "10.0.1.0/24"]) # → false (contained)
provider::burnham::cidrs_are_disjoint(["10.0.0.0/24", "10.0.0.128/25"]) # → false (overlap)
```

**When to use:** `variable` validation on a `list(string)` of subnet CIDRs to ensure no two subnets overlap before creating them — catches mistakes like accidentally including both a summary prefix and a more-specific one in the same list.

```hcl
variable "subnet_cidrs" {
  type = list(string)
  validation {
    condition     = provider::burnham::cidrs_are_disjoint(var.subnet_cidrs)
    error_message = "subnet_cidrs must not contain overlapping entries."
  }
}
```

---

### Info / decomposition

#### `cidr_host_count`

```
cidr_host_count(cidr string) → number
```

Returns the total number of IP addresses in the CIDR (including network and broadcast for IPv4). For very large IPv6 prefixes the result is capped at `MaxInt64`.

```hcl
provider::burnham::cidr_host_count("10.0.0.0/24") # → 256
provider::burnham::cidr_host_count("10.0.0.1/32") # → 1
```

**When to use:** sizing resources based on subnet capacity — asserting a CIDR is large enough to hold the required number of hosts, or computing usable host count (`host_count - 2` for IPv4).

---

#### `cidr_usable_host_count`

```
cidr_usable_host_count(cidr string) → number
```

Returns the number of usable host addresses. For IPv4, subtracts the network and broadcast addresses (`total - 2`), with RFC-correct special cases: `/31` = 2 (point-to-point), `/32` = 1 (host route). For IPv6 all addresses are considered usable.

```hcl
provider::burnham::cidr_usable_host_count("10.0.0.0/24") # → 254
provider::burnham::cidr_usable_host_count("10.0.0.0/31") # → 2   (RFC 3021)
provider::burnham::cidr_usable_host_count("10.0.0.1/32") # → 1
provider::burnham::cidr_usable_host_count("2001:db8::/64") # → 2^64 (capped at MaxInt64)
```

**When to use:** asserting a subnet is large enough to hold a given number of workloads; sizing auto-scaling node pools by available IP space. Saves having to write `cidr_host_count(cidr) - 2` everywhere IPv4 subnets are involved.

---

#### `cidr_first_ip` / `cidr_last_ip`

```
cidr_first_ip(cidr string) → string
cidr_last_ip(cidr string)  → string
```

`cidr_first_ip` returns the network address (all host bits zero); `cidr_last_ip` returns the last address (all host bits set; broadcast for IPv4).

```hcl
provider::burnham::cidr_first_ip("10.0.0.0/24") # → "10.0.0.0"
provider::burnham::cidr_first_ip("10.0.0.7/24") # → "10.0.0.0"  (normalizes)
provider::burnham::cidr_last_ip("10.0.0.0/24")  # → "10.0.0.255"
```

**When to use:** deriving boundary addresses for documentation, DHCP pool configuration, or route policy definitions without hardcoding IPs.

---

#### `cidr_prefix_length`

```
cidr_prefix_length(cidr string) → number
```

Returns the prefix length (`/N`) as a plain number.

```hcl
provider::burnham::cidr_prefix_length("10.0.0.0/23") # → 23
```

**When to use:** resource attributes that expect a plain integer prefix length rather than CIDR notation — BGP route-map configs, some load balancer target group attributes, Kubernetes network policy APIs.

---

#### `cidr_wildcard`

```
cidr_wildcard(cidr string) → string
```

Returns the wildcard mask (bitwise inverse of the subnet mask) for an IPv4 CIDR. Returns an error for IPv6 CIDRs.

```hcl
provider::burnham::cidr_wildcard("10.0.0.0/24") # → "0.0.0.255"
provider::burnham::cidr_wildcard("10.0.0.0/16") # → "0.0.255.255"
```

**When to use:** Cisco IOS / NX-OS ACL entries (`permit ip 10.0.0.0 0.0.0.255 any`), AWS network ACL rules, and any firewall API that uses wildcard mask notation instead of prefix-length.

---

### IP arithmetic

#### `ip_add`

```
ip_add(ip string, n number) → string
```

Returns the IP address offset by `n`. `n` may be negative. Returns an error if the result would overflow the address space. Supports IPv4 and IPv6.

```hcl
provider::burnham::ip_add("10.0.0.0", 1)  # → "10.0.0.1"
provider::burnham::ip_add("10.0.0.5", -3) # → "10.0.0.2"
```

**When to use:** computing specific host addresses from a subnet base — gateway is typically base+1, DNS base+2, NTP base+3. Cleaner than `cidrhost()` when you already have the base IP and just need `+N`.

---

#### `ip_subtract`

```
ip_subtract(a string, b string) → number
```

Returns `a - b` as a signed integer: the number of address positions separating the two IPs. Positive when `a` is higher, negative when `b` is higher. Both addresses must be the same family. For IPv4 the result always fits; for IPv6, returns an error if the high 64 bits differ (difference too large for int64).

```hcl
provider::burnham::ip_subtract("10.0.0.10", "10.0.0.1")  # → 9
provider::burnham::ip_subtract("10.0.0.1", "10.0.0.10")  # → -9
provider::burnham::ip_subtract("10.0.0.5", "10.0.0.5")   # → 0
```

**When to use:** computing range lengths (`ip_subtract(last, first) + 1`); asserting an IP is at the expected offset from a base address; validating that two IPs are within N hops of each other without needing a CIDR context.

---

### Version detection

#### `ip_version` / `cidr_version`

```
ip_version(ip string)    → number  # 4 or 6
cidr_version(cidr string) → number  # 4 or 6
```

Returns the IP version of an address or CIDR. IPv4-mapped IPv6 addresses (e.g. `::ffff:10.0.0.1`) are treated as IPv4.

```hcl
provider::burnham::ip_version("192.168.1.1")  # → 4
provider::burnham::ip_version("2001:db8::1")  # → 6
```

**When to use:** branching logic based on address family when handling mixed IPv4/IPv6 inputs from variables or data sources.

---

#### `cidr_filter_version`

```
cidr_filter_version(cidrs list(string), version number) → list(string)
```

Returns only the CIDRs from the list that belong to the given IP version (4 or 6).

```hcl
provider::burnham::cidr_filter_version(
  ["10.0.0.0/8", "172.16.0.0/12", "2001:db8::/32", "fd00::/8"],
  4
)
# → ["10.0.0.0/8", "172.16.0.0/12"]
```

**When to use:** splitting a dual-stack peer's advertised routes into separate IPv4 and IPv6 route tables; feeding a mixed list from a variable into a resource that only accepts one address family.

---

### Private-range checks

#### `ip_is_private` / `cidr_is_private`

```
ip_is_private(ip string)     → bool
cidr_is_private(cidr string) → bool
```

Returns `true` if the IP or entire CIDR falls within a private, loopback, link-local, or CGNAT range. Covers RFC 1918, RFC 6598 (CGNAT 100.64.0.0/10), RFC 4193 (IPv6 ULA), loopback, and link-local.

```hcl
provider::burnham::ip_is_private("192.168.1.1")  # → true
provider::burnham::ip_is_private("8.8.8.8")      # → false
provider::burnham::ip_is_private("100.64.0.1")   # → true  (CGNAT)
provider::burnham::cidr_is_private("10.0.0.0/8") # → true
```

**When to use:** validating that internal resources use private address space; filtering out RFC 1918 addresses from a public IP allowlist feed.

---

### NAT64 (RFC 6052)

NAT64 allows IPv6-only clients to reach IPv4 servers by translating addresses. A NAT64 prefix (e.g. the Well-Known Prefix `64:ff9b::/96`) is combined with an IPv4 address to synthesize an IPv6 address that the NAT64 gateway knows how to translate back.

Valid NAT64 prefix lengths: `/32`, `/40`, `/48`, `/56`, `/64`, `/96`. The reserved u-octet (bits 64–71) must always be zero.

#### `nat64_prefix_valid`

```
nat64_prefix_valid(prefix string) → bool
```

Returns `true` if the prefix meets RFC 6052 requirements: valid length, IPv6, and u-octet = zero.

```hcl
provider::burnham::nat64_prefix_valid("64:ff9b::/96")    # → true  (Well-Known Prefix)
provider::burnham::nat64_prefix_valid("64:ff9b:1::/48")  # → true  (local-use, RFC 8215)
provider::burnham::nat64_prefix_valid("2001:db8::/44")   # → false (wrong length)
provider::burnham::nat64_prefix_valid("10.0.0.0/24")     # → false (not IPv6)
```

**When to use:** `variable` validation blocks when accepting an operator-supplied NAT64 prefix.

---

#### `nat64_synthesize`

```
nat64_synthesize(ipv4 string, prefix string [, use_hex bool]) → string
```

Embeds an IPv4 address into a NAT64 prefix following the RFC 6052 byte layout. Returns the result in mixed notation (`64:ff9b::192.0.2.1`) by default; pass `true` as the optional third argument for standard hex notation (`64:ff9b::c000:201`).

```hcl
provider::burnham::nat64_synthesize("192.0.2.1", "64:ff9b::/96")
# → "64:ff9b::192.0.2.1"

provider::burnham::nat64_synthesize("192.0.2.1", "64:ff9b::/96", true)
# → "64:ff9b::c000:201"
```

**When to use:** configuring DNS64 AAAA records for IPv4-only services; pre-computing the IPv6 address that IPv6-only clients will use to reach a specific IPv4 endpoint.

---

#### `nat64_extract`

```
nat64_extract(ipv6 string [, nat64_prefix string]) → string
```

Recovers the IPv4 address from a NAT64 IPv6 address.

With no second argument, extracts the last 32 bits directly — correct for any `/96` NAT64 prefix including the Well-Known Prefix `64:ff9b::/96`, and by far the most common case. With an optional `nat64_prefix` argument, uses the RFC 6052 byte layout for that prefix length instead — needed for `/32`–`/64` prefixes where the IPv4 bytes don't sit in the last 32 bits.

```hcl
# Common case — /96, no prefix needed
provider::burnham::nat64_extract("64:ff9b::c000:201")          # → "192.0.2.1"

# Non-/96 prefix — specify it explicitly
provider::burnham::nat64_extract("2001:db8::c0:2:2100:0", "2001:db8::/64")  # → "192.0.2.33"
```

**When to use:** reverse-mapping NAT64 addresses in flow logs or firewall hits back to the original IPv4; ACL generation from IPv6 traffic records.

---

#### `nat64_synthesize_cidrs`

```
nat64_synthesize_cidrs(ipv4_cidrs list(string), nat64_prefix string [, use_hex bool]) → list(string)
```

Converts a list of IPv4 CIDRs to their NAT64 IPv6 CIDR equivalents in one call. Only `/64` and `/96` NAT64 prefixes are supported (the two where IPv4 bits occupy a contiguous range). Returns mixed notation by default.

```hcl
provider::burnham::nat64_synthesize_cidrs(
  ["203.0.113.0/24", "198.51.100.0/24"],
  "64:ff9b::/96"
)
# → ["64:ff9b::203.0.113.0/120", "64:ff9b::198.51.100.0/120"]
```

**When to use:** the primary NAT64 bulk operation. Given an existing IPv4 allowlist, generate the corresponding NAT64 IPv6 ranges and `concat()` them with the IPv4 list to produce a dual-stack security group that works for both IPv4-capable and IPv6-only clients reaching the same services.

```hcl
locals {
  ipv4_allowlist  = ["203.0.113.0/24", "198.51.100.0/24"]
  nat64_allowlist = provider::burnham::nat64_synthesize_cidrs(local.ipv4_allowlist, "64:ff9b::/96")
  full_allowlist  = concat(local.ipv4_allowlist, local.nat64_allowlist)
}
```

---

#### `nat64_synthesize_cidr`

```
nat64_synthesize_cidr(ipv4_cidr string, nat64_prefix string [, use_hex bool]) → string
```

Converts a single IPv4 CIDR to its NAT64 IPv6 CIDR equivalent. Only `/64` and `/96` NAT64 prefixes are supported.

```hcl
provider::burnham::nat64_synthesize_cidr("192.0.2.0/24", "64:ff9b::/96")
# → "64:ff9b::192.0.2.0/120"
```

**When to use:** expressing a single IPv4 pool CIDR in IPv6 form for a route advertisement or NAT64 pool configuration.

---

### NPTv6 (RFC 6296)

NPTv6 provides stateless IPv6-to-IPv6 prefix translation, commonly used when an organization has ULA addresses internally but needs to present a provider-assigned (PA) prefix externally — without the complexity of NAT.

#### `nptv6_translate`

```
nptv6_translate(ipv6 string, from_prefix string, to_prefix string) → string
```

Translates an IPv6 address from one `/48` prefix to another using the RFC 6296 checksum-neutral algorithm. Both prefixes must be `/48`.

The first 48 bits are replaced with the new prefix. An adjustment is applied to bytes 8–9 (the first word of the Interface Identifier) so that the one's complement sum of all 128 bits is preserved — this ensures TCP/UDP checksums remain valid without packet rewriting. A simple prefix swap would produce incorrect addresses.

To reverse a translation, swap `from_prefix` and `to_prefix`.

```hcl
# Internal ULA address → external PA address
provider::burnham::nptv6_translate(
  "fd00::10:0:1",
  "fd00::/48",      # internal (from)
  "2001:db8::/48"   # external (to)
)
# → "2001:db8::xxxx:10:0:1"  (IID adjusted for checksum neutrality)
```

**When to use:** computing the external address a host will appear as through an NPTv6 gateway, for use in DNS records, ACL entries, and route advertisements; reverse-translating external addresses from flow logs back to internal form.

---

### Dual / mixed notation

#### `ip_to_mixed_notation`

```
ip_to_mixed_notation(ip string) → string
```

Formats an IPv6 address using `x:x:x:x:x:x:d.d.d.d` notation, where the last 32 bits are expressed as dotted-decimal. Zero-compression (`::`) is applied to the hex portion. IPv4 addresses are returned unchanged.

```hcl
provider::burnham::ip_to_mixed_notation("64:ff9b::c000:201")
# → "64:ff9b::192.0.2.1"

provider::burnham::ip_to_mixed_notation("::ffff:c0a8:101")
# → "192.168.1.1"  (IPv4-mapped is unmapped to native IPv4)
```

**When to use:** displaying NAT64 addresses in a form that makes the embedded IPv4 immediately visible to operators; formatting addresses for documentation or human-readable outputs.

---

#### `ipv4_to_ipv4_mapped`

```
ipv4_to_ipv4_mapped(ipv4 string) → string
```

Returns the RFC 4291 IPv4-mapped IPv6 representation of an IPv4 address in mixed notation: `::ffff:d.d.d.d`.

```hcl
provider::burnham::ipv4_to_ipv4_mapped("192.0.2.1") # → "::ffff:192.0.2.1"
```

**When to use:** configuring dual-stack sockets or APIs that represent IPv4 connections as IPv4-mapped IPv6 addresses; some BGP implementations, cloud load balancer log formats, and Kubernetes network policies use this notation.

---

### IPAM

#### `cidr_find_free`

```
cidr_find_free(pool list(string), used list(string), prefix_len number) → string
```

Returns the first available prefix of length `prefix_len` within `pool` after removing all `used` CIDRs. Returns `null` if no prefix of that size is available.

```hcl
# First available /24 in a /16, skipping already-allocated subnets
provider::burnham::cidr_find_free(
  ["10.0.0.0/16"],
  ["10.0.0.0/24", "10.0.1.0/24"],
  24
)
# → "10.0.2.0/24"

# Returns null when the pool is exhausted
provider::burnham::cidr_find_free(["10.0.0.0/30"], ["10.0.0.0/30"], 24)
# → null
```

**When to use:** IPAM-style subnet allocation — given a VPC CIDR as the pool and a list of already-allocated subnets as used, find the next free subnet to assign to a new workload. Useful in `locals` blocks to compute the next available AZ subnet without hardcoding offsets.

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

### Pretty-printed JSON

```hcl
locals {
  policy = provider::burnham::jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["s3:GetObject", "s3:ListBucket"]
        Resource = ["arn:aws:s3:::my-bucket/*"]
      },
    ]
  })
  # Output:
  # {
  # 	"Statement": [
  # 		{
  # 			"Action": [
  # 				"s3:GetObject",
  # 				"s3:ListBucket"
  # 			],
  # ...

  # With 2-space indent:
  policy_spaces = provider::burnham::jsonencode({a = 1}, { indent = "  " })
}
```

### HuJSON (JSON with comments and trailing commas)

```hcl
locals {
  # Decode HuJSON — comments stripped, trailing commas handled.
  config = provider::burnham::hujsondecode(file("${path.module}/config.hujson"))

  # Re-encode as HuJSON with comments
  updated = provider::burnham::hujsonencode(
    merge(local.config, { port = 9090 }),
    {
      comments = {
        hosts = "Server hostnames"
        port  = "Main listening port"
        tls   = "Require TLS in production"
      }
    }
  )
}
```

Useful for Tailscale ACL policies or any configuration file where you want comments and trailing commas alongside your JSON. Also parses JSONC files (comments only, no trailing commas) since HuJSON is a superset.

### Decoding a macOS configuration profile

```hcl
locals {
  profile = provider::burnham::plistdecode(file("${path.module}/profile.plist"))

  # Access values naturally
  profile_name = local.profile.PayloadDisplayName
  cache_limit  = local.profile.PayloadContent[0].CacheLimit
}
```

### Building a plist from scratch

```hcl
locals {
  config = provider::burnham::plistencode({
    PayloadDisplayName       = "WiFi - Corporate"
    PayloadIdentifier        = "com.example.wifi"
    PayloadType              = "Configuration"
    PayloadVersion           = 1
    PayloadRemovalDisallowed = true
    PayloadContent = [
      {
        PayloadType  = "com.apple.wifi.managed"
        AutoJoin     = true
        SSID_STR     = "CorpNet"
      },
    ]
  })
}
```

### Modifying a plist (decode, change, re-encode)

Dates, binary data, integer vs real distinction, and all other types are preserved automatically through round-trips.

```hcl
locals {
  original = provider::burnham::plistdecode(file("profile.plist"))
  modified = provider::burnham::plistencode(merge(local.original, {
    PayloadDisplayName = "Updated Name"
  }))
}
```

### Plist with XML comments

```hcl
locals {
  commented_plist = provider::burnham::plistencode(
    {
      PayloadDisplayName = "WiFi - Corporate"
      PayloadIdentifier  = "com.example.wifi"
      PayloadVersion     = 1
    },
    {
      comments = {
        PayloadDisplayName = "Human-readable profile name"
        PayloadIdentifier  = "Unique reverse-DNS identifier"
      }
    }
  )
  # <?xml version="1.0" encoding="UTF-8"?>
  # ...
  # 	<!-- Human-readable profile name -->
  # 	<key>PayloadDisplayName</key>
  # 	<string>WiFi - Corporate</string>
  # 	<!-- Unique reverse-DNS identifier -->
  # 	<key>PayloadIdentifier</key>
  # 	<string>com.example.wifi</string>
  # ...
}
```

### Dates, binary data, and explicit reals in plists

```hcl
locals {
  profile = provider::burnham::plistencode({
    PayloadExpirationDate = provider::burnham::plistdate("2025-12-31T00:00:00Z")
    PayloadContent        = provider::burnham::plistdata(filebase64("${path.module}/cert.der"))
    ScaleFactor           = provider::burnham::plistreal(2) # <real>2</real>, not <integer>2</integer>
  })
  # Produces <date>, <data>, and <real> elements in the plist XML
}
```

### Binary plists

Binary plists aren't valid UTF-8, so use `filebase64()` — Burnham auto-detects the encoding:

```hcl
locals {
  binary = provider::burnham::plistdecode(filebase64("${path.module}/binary.plist"))
}
```

### Nested plists

macOS configuration profiles commonly nest plists inside `<data>` blocks — the outer profile wraps an inner payload as base64-encoded plist data. Build the inner plist with `plistencode`, base64-encode it with Terraform's built-in `base64encode`, and wrap it with `plistdata()`:

```hcl
locals {
  profile = provider::burnham::plistencode({
    PayloadDisplayName = "WiFi"
    PayloadType        = "Configuration"
    PayloadVersion     = 1
    PayloadContent = [
      {
        PayloadType    = "com.apple.wifi.managed"
        PayloadVersion = 1
        PayloadContent = provider::burnham::plistdata(base64encode(
          provider::burnham::plistencode({
            AutoJoin       = true
            SSID_STR       = "CorpNet"
            EncryptionType = "WPA2"
          })
        ))
      },
    ]
  })
}
```

To decode a nested plist, chain `plistdecode` calls — the inner plist is in the tagged data object's `.value`:

```hcl
locals {
  outer = provider::burnham::plistdecode(file("profile.mobileconfig"))
  inner = provider::burnham::plistdecode(local.outer.PayloadContent[0].PayloadContent.value)
  ssid  = local.inner.SSID_STR
}
```

### INI files

```hcl
locals {
  # Decode an INI file
  config = provider::burnham::inidecode(file("${path.module}/config.ini"))
  # => { "" = { ... }, "database" = { "host" = "localhost", "port" = "5432" }, ... }

  db_host = local.config.database.host
  db_port = tonumber(local.config.database.port) # values are always strings

  # Encode an INI file
  new_config = provider::burnham::iniencode({
    database = {
      host = "db.example.com"
      port = "5432"
    }
    cache = {
      enabled = "true"
      ttl     = "3600"
    }
  })
}
```

### CSV encoding

```hcl
locals {
  # Auto-detect headers (sorted alphabetically)
  users_csv = provider::burnham::csvencode([
    { name = "alice", email = "alice@example.com", role = "admin" },
    { name = "bob", email = "bob@example.com", role = "user" },
  ])
  # email,name,role
  # alice@example.com,alice,admin
  # bob@example.com,bob,user

  # Explicit column order
  users_ordered = provider::burnham::csvencode(
    [{ name = "alice", email = "alice@example.com" }],
    { columns = ["name", "email"] }
  )
  # name,email
  # alice,alice@example.com

  # Data only (no header row)
  users_data = provider::burnham::csvencode(
    [{ name = "alice", count = 42, active = true }],
    { columns = ["name", "count", "active"], no_header = true }
  )
  # alice,42,true
}
```

Numbers, bools, and nulls are converted to strings automatically. Commas, quotes, and newlines in values are escaped per RFC 4180.

### YAML (better than built-in)

```hcl
locals {
  # Block style, literal block scalars for scripts — unlike Terraform's yamlencode
  k8s_manifest = provider::burnham::yamlencode({
    apiVersion = "v1"
    kind       = "ConfigMap"
    metadata   = { name = "app-config", namespace = "production" }
    data = {
      "startup.sh" = "#!/bin/bash\nset -e\necho Starting...\n./run-app\n"
    }
  })
  # apiVersion: v1
  # kind: ConfigMap
  # metadata:
  #   name: app-config
  #   namespace: production
  # data:
  #   startup.sh: |
  #     #!/bin/bash
  #     set -e
  #     echo Starting...
  #     ./run-app

  # With comments and options
  annotated = provider::burnham::yamlencode(
    { replicas = 3, image = "nginx:latest" },
    {
      indent = 4
      comments = {
        replicas = "Desired pod count"
        image    = "Container image"
      }
    }
  )
  # # Desired pod count
  # replicas: 3
  # # Container image
  # image: nginx:latest

  # Deduplicate identical subtrees with YAML anchors
  deduped = provider::burnham::yamlencode(
    {
      dev     = { db = { host = "localhost", port = 5432 } }
      staging = { db = { host = "localhost", port = 5432 } }
      prod    = { db = { host = "db.prod.internal", port = 5432 } }
    },
    { dedupe = true }
  )
  # dev: &_ref1
  #   db: ...
  # staging: *_ref1
  # prod:
  #   db: ...
}
```

### Windows .reg files

```hcl
locals {
  # Decode a .reg file
  reg = provider::burnham::regdecode(file("${path.module}/settings.reg"))
  app_name = local.reg["HKEY_LOCAL_MACHINE\\SOFTWARE\\MyApp"].DisplayName

  # Build a .reg file with comments
  new_reg = provider::burnham::regencode(
    {
      "HKEY_LOCAL_MACHINE\\SOFTWARE\\MyApp" = {
        "DisplayName" = "My Application"
        "Version"     = provider::burnham::regdword(2)
        "InstallPath" = provider::burnham::regexpandsz("%ProgramFiles%\\MyApp")
        "Features"    = provider::burnham::regmulti(["core", "plugins", "updates"])
      }
    },
    {
      comments = {
        "HKEY_LOCAL_MACHINE\\SOFTWARE\\MyApp" = {
          "Version"     = "Incremented on each release"
          "InstallPath" = "Uses %ProgramFiles% for standard location"
        }
      }
    }
  )
}
```

### Valve Data Format (VDF)

```hcl
locals {
  # Decode a Steam config file
  library = provider::burnham::vdfdecode(file("${path.module}/libraryfolders.vdf"))
  steam_path = local.library.libraryfolders["0"].path

  # Build a VDF config
  config = provider::burnham::vdfencode({
    AppState = {
      appid      = "730"
      name       = "Counter-Strike 2"
      installdir = "Counter-Strike Global Offensive"
      UserConfig = {
        language = "english"
      }
    }
  })
}
```

### KDL

```hcl
locals {
  # Decode a KDL document
  doc = provider::burnham::kdldecode(<<-EOT
    title "My Config"
    server "web" host="0.0.0.0" port=8080 {
      tls enabled=true
    }
  EOT
  )
  title = local.doc[0].args[0] # "My Config"

  # Encode a KDL document (v2 default, v1 available)
  config = provider::burnham::kdlencode([
    { name = "title", args = ["My Config"], props = {}, children = [] },
    { name = "server", args = ["web"], props = { host = "0.0.0.0", port = 8080 }, children = [
      { name = "tls", args = [], props = { enabled = true }, children = [] },
    ]},
  ])
}
```

### Dual-stack security group from an IPv4 allowlist

Take an existing IPv4 allowlist and produce the corresponding NAT64 IPv6 ranges, then merge both into a single rule set. IPv6-only clients can reach the same services through a NAT64 gateway with no extra configuration.

```hcl
locals {
  ipv4_allow = ["203.0.113.0/24", "198.51.100.0/24"]
  ipv6_allow = provider::burnham::nat64_synthesize_cidrs(local.ipv4_allow, "64:ff9b::/96")
  full_allow = concat(local.ipv4_allow, local.ipv6_allow)
}
```

### Compact a sprawling allowlist before pushing it to a cloud rule limit

Cloud providers cap the number of rules per security group. Merging redundant prefixes before applying avoids hitting those limits.

```hcl
locals {
  raw_cidrs = [
    "10.0.0.0/24", "10.0.1.0/24", "10.0.2.0/23",   # → "10.0.0.0/22"
    "10.0.4.0/24", "10.0.5.0/24",                  # → "10.0.4.0/23"
    "192.168.1.0/24", "192.168.1.0/25",            # /25 is redundant
  ]
  merged = provider::burnham::cidr_merge(local.raw_cidrs)
  # → ["10.0.0.0/22", "10.0.4.0/23", "192.168.1.0/24"]
}
```

### Validate non-overlapping subnet inputs

Catch operator mistakes — accidentally including a summary prefix and a more-specific one in the same list — at plan time instead of at API call time.

```hcl
variable "subnet_cidrs" {
  type = list(string)
  validation {
    condition     = provider::burnham::cidrs_are_disjoint(var.subnet_cidrs)
    error_message = "subnet_cidrs must not contain overlapping entries."
  }
}
```

### Find the next free /24 in an IPAM pool

```hcl
locals {
  vpc_cidrs       = ["10.0.0.0/16"]
  allocated       = ["10.0.0.0/24", "10.0.1.0/24", "10.0.3.0/24"]
  next_free_block = provider::burnham::cidr_find_free(local.vpc_cidrs, local.allocated, 24)
  # → "10.0.2.0/24"
}
```

See [`examples/main.tf`](examples/main.tf) for a complete working example of every function in both families.

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

Then run `terraform plan` or `terraform console` in the `examples/` directory — no `terraform init` needed with dev overrides.

## License

MPL-2.0
