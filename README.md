# Terraform Provider Burnham

> *"Aim high in hope and work, remembering that a noble, logical diagram once recorded will never die."*
> — Daniel Burnham

In 1909, Daniel Burnham published the [Plan of Chicago](https://en.wikipedia.org/wiki/Burnham_Plan_of_Chicago) — a comprehensive blueprint that transformed a sprawling, chaotic city into something coherent and enduring. He believed that good planning wasn't just about what you build, but about making the plan itself clear, readable, and maintainable for generations to come.

Terraform plans deserve the same treatment. But today, when your Terraform needs to work with structured data formats like property lists, human-edited JSON, or pretty-printed configuration files, you're stuck with workarounds — shelling out to external tools, embedding raw strings, or losing type fidelity in translation. The plan becomes cluttered and fragile.

Burnham fixes this. It's a pure function provider — no resources, no data sources, no API calls — that gives Terraform native fluency with the data formats it can't handle cleanly on its own.

| Format | Encode | Decode | Notes |
|--------|--------|--------|-------|
| JSON (pretty-printed) | `jsonencode` | — | Terraform has `jsondecode` built-in |
| HuJSON / JWCC | `hujsonencode` | `hujsondecode` | JSON with comments and trailing commas |
| Apple Property List | `plistencode` | `plistdecode` | XML, binary, and OpenStep formats |
| INI | `iniencode` | `inidecode` | Standard `[section]` / `key = value` files |
| TOML | — | — | Use [Tobotimus/toml](https://registry.terraform.io/providers/Tobotimus/toml) instead |

Your configuration profiles, ACL policies, and structured documents become first-class citizens in your Terraform plans, not opaque blobs passed through `file()` and hoped for the best.

The result is Terraform code that reads like a blueprint — clear, logical, and built to last.

## Function Reference

### `jsonencode`

Encode a value as pretty-printed JSON. Unlike Terraform's built-in `jsonencode`, this produces human-readable output with configurable indentation.

```
provider::burnham::jsonencode(value, indent?) → string
```

| Parameter | Type | Required | Description |
|---|---|---|---|
| `value` | `dynamic` | Yes | Any Terraform value to encode as JSON. |
| `indent` | `string` | No | Indentation string for each level. Default: `"\t"` (tab). |

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

Encode a Terraform value as a HuJSON string with trailing commas and pretty-printed formatting.

```
provider::burnham::hujsonencode(value, indent?) → string
```

| Parameter | Type | Required | Description |
|---|---|---|---|
| `value` | `dynamic` | Yes | Any Terraform value to encode. |
| `indent` | `string` | No | Indentation string for each level. Default: `"\t"` (tab). |

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
provider::burnham::plistencode(value, format?) → string
```

| Parameter | Type | Required | Description |
|---|---|---|---|
| `value` | `dynamic` | Yes | The value to encode. Tagged objects from `plistdate()` and `plistdata()` are converted to native `<date>` and `<data>` elements. |
| `format` | `string` | No | Output format: `"xml"` (default), `"binary"`, or `"openstep"`. |

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
  policy_spaces = provider::burnham::jsonencode({a = 1}, "  ")
}
```

### HuJSON (JSON with comments and trailing commas)

```hcl
locals {
  # Decode HuJSON — comments stripped, trailing commas handled.
  # Works with any JWCC/HuJSON file: Tailscale ACLs, annotated configs, etc.
  config = provider::burnham::hujsondecode(file("${path.module}/config.hujson"))

  # Modify and re-encode as HuJSON (trailing commas + formatting added)
  updated = provider::burnham::hujsonencode(merge(local.config, {
    port = 9090
  }))
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

See [`examples/main.tf`](examples/main.tf) for a complete working example of all functions.

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
