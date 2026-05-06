terraform {
  required_providers {
    burnham = {
      source = "keeleysam/burnham"
    }
  }
}

# ─── jsonencode ────────────────────────────────────────────────────
# Pretty-prints any Terraform value as JSON. Default indent is tab.

output "json_tabs" {
  description = "Pretty-printed JSON with tab indentation (default)"
  value = provider::burnham::jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["s3:GetObject", "s3:ListBucket"]
        Resource = ["arn:aws:s3:::my-bucket", "arn:aws:s3:::my-bucket/*"]
      },
    ]
  })
}

output "json_spaces" {
  description = "Pretty-printed JSON with 2-space indentation"
  value       = provider::burnham::jsonencode({ count = 42, enabled = true, name = "test" }, { indent = "  " })
}

# ─── hujsondecode ─────────────────────────────────────────────────
# Parses HuJSON (JSON with // comments, /* block comments */, and trailing commas).
# Also accepts standard JSON.

locals {
  hujson_input = <<-EOT
    {
      // Server configuration
      "hosts": [
        "app-1.example.com",
        "app-2.example.com",
      ],
      "port": 8080,
      /* TLS is required in production */
      "tls": true,
    }
  EOT

  decoded_hujson = provider::burnham::hujsondecode(local.hujson_input)
}

output "hujson_decoded_hosts" {
  description = "Accessing values from decoded HuJSON"
  value       = local.decoded_hujson.hosts
}

output "hujson_decoded_port" {
  description = "Numeric value from decoded HuJSON"
  value       = local.decoded_hujson.port
}

# ─── hujsonencode ─────────────────────────────────────────────────
# Encodes a value as HuJSON with trailing commas. Default indent is tab.

output "hujson_encode_tabs" {
  description = "HuJSON with tab indentation (default)"
  value       = provider::burnham::hujsonencode(local.decoded_hujson)
}

output "hujson_encode_spaces" {
  description = "HuJSON with 2-space indentation"
  value       = provider::burnham::hujsonencode(local.decoded_hujson, { indent = "  " })
}

output "hujson_with_comments" {
  description = "HuJSON with declarative comments"
  value = provider::burnham::hujsonencode(
    local.decoded_hujson,
    {
      comments = {
        hosts = "Server hostnames"
        port  = "Main listening port"
        tls   = "Require TLS in production"
      }
    }
  )
}

# ─── plistdecode ──────────────────────────────────────────────────
# Parses Apple property lists. Auto-detects XML, binary, and OpenStep formats.
# Also auto-detects base64-encoded input (for binary plists via filebase64()).

locals {
  plist_xml = <<-EOT
    <?xml version="1.0" encoding="UTF-8"?>
    <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
    <plist version="1.0">
    <dict>
      <key>PayloadDisplayName</key>
      <string>Content Caching</string>
      <key>PayloadVersion</key>
      <integer>1</integer>
      <key>PayloadEnabled</key>
      <true/>
      <key>Rating</key>
      <real>4.5</real>
      <key>PayloadContent</key>
      <array>
        <dict>
          <key>AllowSharedCaching</key>
          <true/>
          <key>CacheLimit</key>
          <integer>0</integer>
        </dict>
      </array>
    </dict>
    </plist>
  EOT

  decoded_plist = provider::burnham::plistdecode(local.plist_xml)
}

output "plist_string_value" {
  description = "Accessing a string from a decoded plist"
  value       = local.decoded_plist.PayloadDisplayName
}

output "plist_integer_value" {
  description = "Accessing an integer from a decoded plist"
  value       = local.decoded_plist.PayloadVersion
}

output "plist_bool_value" {
  description = "Accessing a bool from a decoded plist"
  value       = local.decoded_plist.PayloadEnabled
}

output "plist_float_value" {
  description = "Accessing a fractional real from a decoded plist (plain number)"
  value       = local.decoded_plist.Rating
}

output "plist_nested_value" {
  description = "Accessing a nested value from a decoded plist"
  value       = local.decoded_plist.PayloadContent[0].CacheLimit
}

# Decode a plist that has <date> and <data> elements
locals {
  plist_with_special_types = <<-EOT
    <?xml version="1.0" encoding="UTF-8"?>
    <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
    <plist version="1.0">
    <dict>
      <key>ExpirationDate</key>
      <date>2025-12-31T00:00:00Z</date>
      <key>Certificate</key>
      <data>SGVsbG8gV29ybGQ=</data>
      <key>WholeReal</key>
      <real>2</real>
    </dict>
    </plist>
  EOT

  decoded_special = provider::burnham::plistdecode(local.plist_with_special_types)
}

output "plist_date_tagged_object" {
  description = "A <date> element decodes as a tagged object"
  value       = local.decoded_special.ExpirationDate
  # => { __plist_type = "date", value = "2025-12-31T00:00:00Z" }
}

output "plist_date_value" {
  description = "Accessing the RFC 3339 string from a decoded date"
  value       = local.decoded_special.ExpirationDate.value
  # => "2025-12-31T00:00:00Z"
}

output "plist_data_tagged_object" {
  description = "A <data> element decodes as a tagged object"
  value       = local.decoded_special.Certificate
  # => { __plist_type = "data", value = "SGVsbG8gV29ybGQ=" }
}

output "plist_data_base64" {
  description = "Accessing the base64 string from decoded binary data"
  value       = local.decoded_special.Certificate.value
  # => "SGVsbG8gV29ybGQ="
}

output "plist_real_tagged_object" {
  description = "A whole-number <real> decodes as a tagged object to distinguish from <integer>"
  value       = local.decoded_special.WholeReal
  # => { __plist_type = "real", value = "2" }
}

# ─── plistencode ──────────────────────────────────────────────────
# Encodes a value as an Apple property list. Default format is XML.

output "plist_encode_xml" {
  description = "Build a plist from scratch (XML, default)"
  value = provider::burnham::plistencode({
    PayloadDisplayName       = "WiFi - Corporate"
    PayloadIdentifier        = "com.example.wifi"
    PayloadType              = "Configuration"
    PayloadVersion           = 1
    PayloadRemovalDisallowed = true
    PayloadContent = [
      {
        PayloadType    = "com.apple.wifi.managed"
        AutoJoin       = true
        HIDDEN_NETWORK = false
        SSID_STR       = "CorpNet"
        EncryptionType = "WPA2"
      },
    ]
  })
}

output "plist_encode_openstep" {
  description = "Encode as OpenStep/GNUStep format"
  value = provider::burnham::plistencode({
    Name    = "OpenStep Example"
    Version = 1
  }, { format = "openstep" })
}

output "plist_encode_binary_b64" {
  description = "Encode as binary plist (output is base64 since Terraform strings are UTF-8)"
  value = provider::burnham::plistencode({
    Name    = "Binary Example"
    Version = 1
  }, { format = "binary" })
}

# ─── plistencode with XML comments ────────────────────────────────

output "plist_with_comments" {
  description = "Plist XML with <!-- comments -->"
  value = provider::burnham::plistencode(
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
}

# ─── plistdate ────────────────────────────────────────────────────
# Creates a tagged date object for use in plistencode.

output "plist_with_date" {
  description = "Using plistdate() to create a <date> element"
  value = provider::burnham::plistencode({
    PayloadExpirationDate = provider::burnham::plistdate("2025-12-31T00:00:00Z")
    PayloadIdentifier     = "com.example.expiring"
  })
}

# ─── plistdata ────────────────────────────────────────────────────
# Creates a tagged data object for use in plistencode.

output "plist_with_data" {
  description = "Using plistdata() to create a <data> element"
  value = provider::burnham::plistencode({
    PayloadContent = provider::burnham::plistdata("SGVsbG8gV29ybGQ=")
  })
}

# ─── plistreal ────────────────────────────────────────────────────
# Creates a tagged real object to force <real> instead of <integer> for whole numbers.

output "plist_with_real" {
  description = "Using plistreal() to force <real>2</real> instead of <integer>2</integer>"
  value = provider::burnham::plistencode({
    ScaleFactor = provider::burnham::plistreal(2)
    Ratio       = 3.14 # fractional numbers are automatically <real>
    Count       = 2    # plain integers are <integer>
  })
}

# ─── Nested plists ────────────────────────────────────────────────
# macOS configuration profiles commonly nest plists inside <data> blocks.
# Build the inner plist, base64-encode it, and wrap with plistdata().

output "nested_plist" {
  description = "A configuration profile with a nested plist payload inside a <data> block"
  value = provider::burnham::plistencode({
    PayloadDisplayName = "WiFi (Nested)"
    PayloadIdentifier  = "com.example.wifi"
    PayloadType        = "Configuration"
    PayloadVersion     = 1
    PayloadContent = [
      {
        PayloadType       = "com.apple.wifi.managed"
        PayloadIdentifier = "com.example.wifi.payload"
        PayloadVersion    = 1
        # The inner plist is encoded and wrapped as binary data
        PayloadContent = provider::burnham::plistdata(base64encode(
          provider::burnham::plistencode({
            AutoJoin           = true
            HIDDEN_NETWORK     = false
            SSID_STR           = "CorpNet"
            EncryptionType     = "WPA2"
            ProxyType          = "None"
          })
        ))
      },
    ]
  })
}

# ─── inidecode ────────────────────────────────────────────────────
# Parses INI files. All values are strings. Global keys go under "".

locals {
  ini_input = <<-EOT
    app_name = My Application

    [database]
    host = localhost
    port = 5432
    name = mydb

    [cache]
    enabled = true
    ttl = 3600
  EOT

  decoded_ini = provider::burnham::inidecode(local.ini_input)
}

output "ini_app_name" {
  description = "Global key from decoded INI"
  value       = local.decoded_ini[""].app_name
}

output "ini_db_host" {
  description = "Section key from decoded INI"
  value       = local.decoded_ini.database.host
}

output "ini_db_port_as_number" {
  description = "INI values are strings — convert with tonumber() if needed"
  value       = tonumber(local.decoded_ini.database.port)
}

# ─── iniencode ────────────────────────────────────────────────────
# Encodes a Terraform object as an INI file.

output "ini_encoded" {
  description = "Encode an INI file from a Terraform object"
  value = provider::burnham::iniencode({
    database = {
      host = "db.example.com"
      port = "5432"
      name = "production"
    }
    cache = {
      enabled = "true"
      ttl     = "3600"
    }
  })
}

# INI round-trip
output "ini_roundtrip" {
  description = "Decode an INI, re-encode — structure is preserved"
  value       = provider::burnham::iniencode(local.decoded_ini)
}

# ─── csvencode ────────────────────────────────────────────────────
# Encodes a list of objects as CSV. Terraform has csvdecode but no csvencode.

output "csv_auto_headers" {
  description = "CSV with auto-detected headers (sorted alphabetically)"
  value = provider::burnham::csvencode([
    { name = "alice", email = "alice@example.com", role = "admin" },
    { name = "bob", email = "bob@example.com", role = "user" },
  ])
}

output "csv_explicit_columns" {
  description = "CSV with explicit column order"
  value = provider::burnham::csvencode(
    [
      { name = "alice", email = "alice@example.com", role = "admin" },
      { name = "bob", email = "bob@example.com", role = "user" },
    ],
    { columns = ["name", "email", "role"] }
  )
}

output "csv_no_header" {
  description = "CSV data only, no header row"
  value = provider::burnham::csvencode(
    [{ name = "alice", count = 42, active = true }],
    { columns = ["name", "count", "active"], no_header = true }
  )
}

output "csv_type_coercion" {
  description = "Numbers, bools, and nulls are converted to strings (CSV has no type system)"
  value = provider::burnham::csvencode([
    { name = "alice", count = 42, ratio = 3.14, active = true },
    { name = "bob", count = 0, ratio = 1.0, active = false },
  ])
}

# ─── yamlencode ───────────────────────────────────────────────────
# Better YAML encoding: block style, literal block scalars, comments.

output "yaml_k8s_configmap" {
  description = "Kubernetes ConfigMap with multi-line script (the killer use case)"
  value = provider::burnham::yamlencode({
    apiVersion = "v1"
    kind       = "ConfigMap"
    metadata   = { name = "app-config", namespace = "production" }
    data = {
      "startup.sh" = "#!/bin/bash\nset -e\necho Starting...\n./run-app\n"
      "config.yml" = "server:\n  port: 8080\n  host: 0.0.0.0\n"
    }
  })
}

output "yaml_with_comments" {
  description = "YAML with comments"
  value = provider::burnham::yamlencode(
    {
      apiVersion = "v1"
      kind       = "Deployment"
      metadata   = { name = "web", namespace = "production" }
    },
    {
      comments = {
        apiVersion = "Kubernetes API version"
        kind       = "Resource type"
        metadata = {
          name      = "Deployment name"
          namespace = "Target namespace"
        }
      }
    }
  )
}

output "yaml_with_options" {
  description = "YAML with custom formatting options"
  value = provider::burnham::yamlencode(
    { name = "test", items = ["a", "b", "c"], enabled = true },
    {
      indent      = 4
      quote_style = "double"
      null_value  = "~"
    }
  )
}

output "yaml_dedupe" {
  description = "Identical subtrees are deduped with YAML anchors and aliases"
  value = provider::burnham::yamlencode(
    {
      development = {
        database = { adapter = "postgres", host = "localhost", port = 5432 }
        cache    = { adapter = "redis", host = "localhost", port = 6379 }
      }
      staging = {
        database = { adapter = "postgres", host = "localhost", port = 5432 }
        cache    = { adapter = "redis", host = "localhost", port = 6379 }
      }
      production = {
        database = { adapter = "postgres", host = "db.prod.internal", port = 5432 }
        cache    = { adapter = "redis", host = "cache.prod.internal", port = 6379 }
      }
    },
    { dedupe = true }
  )
}

# ─── kdldecode / kdlencode ────────────────────────────────────────
# Parse and generate KDL (KDL Document Language) — a modern config format.

output "kdl_decoded" {
  description = "Decode a KDL document into node objects"
  value = provider::burnham::kdldecode(<<-EOT
    title "My Application"
    server "web" host="0.0.0.0" port=8080 {
      tls enabled=true
    }
  EOT
  )
}

output "kdl_title" {
  description = "Access the first argument of the first node"
  value       = provider::burnham::kdldecode("title \"Hello\"")[0].args[0]
}

output "kdl_encoded_v2" {
  description = "Encode as KDL v2 (default)"
  value = provider::burnham::kdlencode([
    { name = "title", args = ["My App"], props = {}, children = [] },
    { name = "server", args = ["web"], props = { host = "0.0.0.0", port = 8080 }, children = [
      { name = "tls", args = [], props = { enabled = true }, children = [] },
    ]},
  ])
}

output "kdl_encoded_v1" {
  description = "Encode as KDL v1"
  value = provider::burnham::kdlencode(
    [{ name = "config", args = ["value"], props = { key = "test" }, children = [] }],
    { version = "v1" }
  )
}

# ─── vdfdecode / vdfencode ────────────────────────────────────────
# Parse and generate Valve Data Format (VDF) files — used by Steam and Source engine.

output "vdf_decoded" {
  description = "Decode a VDF string (Steam/Source engine config format)"
  value = provider::burnham::vdfdecode(<<-EOT
    "libraryfolders"
    {
      "0"
      {
        "path"    "/Applications/Steam"
        "label"   ""
        // Game sizes
        "apps"
        {
          "730"     "26685592507"
          "440"     "21899556124"
        }
      }
    }
  EOT
  )
}

output "vdf_steam_path" {
  description = "Accessing a value from decoded VDF"
  value       = provider::burnham::vdfdecode("\"Config\"\n{\n\t\"path\"\t\t\"/Applications/Steam\"\n}\n").Config.path
}

output "vdf_encoded" {
  description = "Encode a Terraform object as VDF"
  value = provider::burnham::vdfencode({
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

# ─── regdecode / regencode ────────────────────────────────────────
# Parse and generate Windows .reg (Registry Editor export) files.

locals {
  reg_input = <<-EOT
    Windows Registry Editor Version 5.00

    [HKEY_LOCAL_MACHINE\SOFTWARE\MyApp]
    "DisplayName"="My Application"
    "Version"=dword:00000002
    "Data"=hex:48,65,6c,6c,6f
    @="Default Value"
  EOT

  decoded_reg = provider::burnham::regdecode(local.reg_input)
}

output "reg_decoded" {
  description = "Decode a .reg file — REG_SZ becomes strings, other types use tagged objects"
  value       = local.decoded_reg
}

output "reg_string_value" {
  description = "Accessing a REG_SZ string from decoded .reg"
  value       = local.decoded_reg["HKEY_LOCAL_MACHINE\\SOFTWARE\\MyApp"].DisplayName
}

output "reg_dword_value" {
  description = "Accessing a REG_DWORD value (tagged object)"
  value       = local.decoded_reg["HKEY_LOCAL_MACHINE\\SOFTWARE\\MyApp"].Version
  # => { __reg_type = "dword", value = "2" }
}

output "reg_default_value" {
  description = "Accessing the default value (@)"
  value       = local.decoded_reg["HKEY_LOCAL_MACHINE\\SOFTWARE\\MyApp"]["@"]
}

output "reg_encode_all_types" {
  description = "Build a .reg file with all supported value types"
  value = provider::burnham::regencode({
    "HKEY_LOCAL_MACHINE\\SOFTWARE\\MyApp" = {
      "DisplayName" = "My Application"
      "Version"     = provider::burnham::regdword(2)
      "BigNumber"   = provider::burnham::regqword(1099511627776)
      "Data"        = provider::burnham::regbinary("48656c6c6f")
      "InstallPath" = provider::burnham::regexpandsz("%ProgramFiles%\\MyApp")
      "Features"    = provider::burnham::regmulti(["core", "plugins", "updates"])
      "@"           = "Default Value"
    }
  })
}

output "reg_encode_with_comments" {
  description = "Build a .reg file with ; comments"
  value = provider::burnham::regencode(
    {
      "HKEY_LOCAL_MACHINE\\SOFTWARE\\MyApp" = {
        "DisplayName" = "My Application"
        "Version"     = provider::burnham::regdword(2)
      }
    },
    {
      comments = {
        "HKEY_LOCAL_MACHINE\\SOFTWARE\\MyApp" = {
          "DisplayName" = "Human-readable application name"
          "Version"     = "Incremented on each release"
        }
      }
    }
  )
}

# ─── Round-trip: decode → modify → re-encode ─────────────────────
# All types (dates, data, integer vs real) are preserved automatically.

output "plist_roundtrip" {
  description = "Decode a plist, modify it, re-encode — types are preserved"
  value = provider::burnham::plistencode(merge(local.decoded_special, {
    ExpirationDate = provider::burnham::plistdate("2026-06-01T00:00:00Z")
  }))
}

# ═══════════════════════════════════════════════════════════════════
#                          Networking
# ═══════════════════════════════════════════════════════════════════

# ─── CIDR set operations ─────────────────────────────────────────

output "cidr_merge" {
  description = "Aggregate adjacent and redundant CIDRs"
  value = provider::burnham::cidr_merge([
    "10.0.0.0/24", "10.0.1.0/24", # → "10.0.0.0/23"
    "10.0.0.0/25",                # redundant
    "192.168.1.0/24",
  ])
}

output "cidr_subtract" {
  description = "Carve a reserved range out of an allocation"
  value       = provider::burnham::cidr_subtract(["10.0.0.0/8"], ["10.1.0.0/16"])
}

output "cidr_intersect" {
  description = "Address space common to two lists"
  value = provider::burnham::cidr_intersect(
    ["10.0.0.0/8", "172.16.0.0/12"],
    ["10.100.0.0/16", "192.168.0.0/16"],
  )
}

output "range_to_cidrs" {
  description = "Convert an inclusive IP range (e.g. cloud feed) to CIDRs"
  value       = provider::burnham::range_to_cidrs("10.0.0.1", "10.0.0.6")
}

output "cidr_enumerate" {
  description = "All /26 subnets within a /24"
  value       = provider::burnham::cidr_enumerate("10.0.0.0/24", 2)
}

# ─── Queries / validation ────────────────────────────────────────

output "ip_in_cidr" {
  description = "Is the bastion address inside the management subnet?"
  value       = provider::burnham::ip_in_cidr("10.0.1.50", "10.0.1.0/24")
}

output "cidr_contains" {
  description = "Is one CIDR fully nested inside another?"
  value       = provider::burnham::cidr_contains("10.0.0.0/8", "10.1.2.0/24")
}

output "cidrs_are_disjoint" {
  description = "Validate a list of subnet CIDRs has no overlaps"
  value       = provider::burnham::cidrs_are_disjoint(["10.0.0.0/24", "10.0.1.0/24"])
}

# ─── Decomposition / arithmetic ──────────────────────────────────

locals {
  mgmt_subnet = "10.0.1.0/24"
  mgmt_first  = provider::burnham::cidr_first_ip(local.mgmt_subnet)
}

output "subnet_gateway" {
  description = "Conventional gateway address: subnet base + 1"
  value       = provider::burnham::ip_add(local.mgmt_first, 1)
}

output "subnet_dns" {
  description = "Conventional DNS address: subnet base + 2"
  value       = provider::burnham::ip_add(local.mgmt_first, 2)
}

output "subnet_usable_hosts" {
  description = "Usable hosts in a /24 (excludes network + broadcast)"
  value       = provider::burnham::cidr_usable_host_count("10.0.1.0/24")
}

output "ipv4_cisco_wildcard" {
  description = "Wildcard mask for Cisco-style ACL entries"
  value       = provider::burnham::cidr_wildcard("10.0.0.0/24")
}

# ─── Private-range checks ────────────────────────────────────────

output "is_private_192" {
  description = "RFC 1918"
  value       = provider::burnham::ip_is_private("192.168.1.1")
}

output "is_private_cgnat" {
  description = "RFC 6598 CGNAT"
  value       = provider::burnham::ip_is_private("100.64.0.1")
}

output "is_private_8888" {
  description = "Public — Google DNS"
  value       = provider::burnham::ip_is_private("8.8.8.8")
}

# ─── Dual-stack: NAT64 synthesis from an IPv4 allowlist ──────────

locals {
  ipv4_allowlist = ["203.0.113.0/24", "198.51.100.0/24"]
  nat64_prefix   = "64:ff9b::/96"
  ipv6_allowlist = provider::burnham::nat64_synthesize_cidrs(local.ipv4_allowlist, local.nat64_prefix)
  full_allowlist = concat(local.ipv4_allowlist, local.ipv6_allowlist)
}

output "nat64_full_allowlist" {
  description = "Original IPv4 ranges plus their NAT64 IPv6 equivalents"
  value       = local.full_allowlist
}

output "nat64_synthesize_single_host" {
  description = "Single-host NAT64 synthesis in mixed notation"
  value       = provider::burnham::nat64_synthesize("192.0.2.1", "64:ff9b::/96")
}

output "nat64_extract_roundtrip" {
  description = "Recover the IPv4 from a NAT64 IPv6 address"
  value       = provider::burnham::nat64_extract("64:ff9b::192.0.2.1")
}

# ─── NPTv6: ULA → public prefix translation ──────────────────────

output "nptv6_translated" {
  description = "Translate a ULA address to its checksum-neutral PA equivalent"
  value = provider::burnham::nptv6_translate(
    "fd00::10:0:1",
    "fd00::/48",
    "2001:db8::/48",
  )
}

# ─── IPAM: find the next free subnet ─────────────────────────────

output "next_free_subnet" {
  description = "Next available /24 in the VPC pool, skipping allocated blocks"
  value = provider::burnham::cidr_find_free(
    ["10.0.0.0/16"],
    ["10.0.0.0/24", "10.0.1.0/24"],
    24,
  )
}
