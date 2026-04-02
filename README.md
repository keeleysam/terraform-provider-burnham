# Terraform Provider Burnham

<p align="center">
  <img src="assets/logo.svg" alt="Burnham" width="300" height="300">
</p>

> *"Aim high in hope and work, remembering that a noble, logical diagram once recorded will never die."*
> — Daniel Burnham

In 1909, Daniel Burnham published the [Plan of Chicago](https://en.wikipedia.org/wiki/Burnham_Plan_of_Chicago) — a comprehensive blueprint that transformed a sprawling, chaotic city into something coherent and enduring. He believed that good planning wasn't just about what you build, but about making the plan itself clear, readable, and maintainable for generations to come.

Terraform plans deserve the same treatment. But today, when your Terraform needs to work with structured data formats like property lists, human-edited JSON, or pretty-printed configuration files, you're stuck with workarounds — shelling out to external tools, embedding raw strings, or losing type fidelity in translation. The plan becomes cluttered and fragile.

Burnham fixes this. It's a pure function provider — no resources, no data sources, no API calls — that gives Terraform native fluency with the data formats it can't handle cleanly on its own.

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
| TOML | — | — | Use [Tobotimus/toml](https://registry.terraform.io/providers/Tobotimus/toml) instead |

Your configuration profiles, ACL policies, and structured documents become first-class citizens in your Terraform plans, not opaque blobs passed through `file()` and hoped for the best.

The result is Terraform code that reads like a blueprint — clear, logical, and built to last.

## Function Reference

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
