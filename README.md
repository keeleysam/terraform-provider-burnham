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

Per-function documentation — including parameters, options, and return values — lives under [`docs/functions/`](docs/functions/) and on [registry.terraform.io](https://registry.terraform.io/providers/keeleysam/burnham/latest/docs). The pages there are auto-generated from the function metadata in source, so they always match the latest published version.

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

A short tour. See [`examples/`](examples/) for the full set of working snippets — every function has at least one — and [`docs/functions/`](docs/functions/) for per-function reference.

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

  # Single call, both families collapse independently:
  merged = provider::burnham::cidr_merge(concat(local.ipv4_blocks, local.ipv6_blocks))
  # → ["10.0.0.0/22", "2001:db8::/64", "2001:db9::/64"]
}
```

For the canonical "build a NAT64-aware dual-stack allowlist from an existing IPv4 list" pattern, see [`examples/networking.tf`](examples/networking.tf).

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
