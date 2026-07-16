Parses an [Apple property list](https://developer.apple.com/documentation/foundation/archives_and_serialization/property_lists) string into a Terraform value.

The format is auto-detected: XML, binary, OpenStep, and GNUStep are all accepted. For binary plists, pass the output of `filebase64()`; base64-encoded input is detected automatically.

Most elements decode to plain Terraform values. Three types decode as tagged objects so they round-trip cleanly back through `plistencode`:

- `<date>` becomes `{ __plist_type = "date", value = "<RFC 3339 string>" }`
- `<data>` becomes `{ __plist_type = "data", value = "<base64>" }`
- a whole-number `<real>` becomes `{ __plist_type = "real", value = "..." }`, distinguishing it from `<integer>`

**Common uses:** reading Apple configuration profiles (`.mobileconfig`), `.plist` preference files, or any payload from MDM tooling where the on-disk format may be XML or binary.