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
