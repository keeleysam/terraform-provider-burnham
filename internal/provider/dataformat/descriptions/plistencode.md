Encodes a Terraform value as an [Apple property list](https://developer.apple.com/documentation/foundation/archives_and_serialization/property_lists) string.

The `format` key in `options` selects the output format:

- `xml` (default): the standard XML plist
- `binary`: a base64-encoded binary plist
- `openstep`: the OpenStep/GNUStep textual format

Tagged objects from `plistdate()`, `plistdata()`, and `plistreal()` are converted to native `<date>`, `<data>`, and `<real>` elements. A number with no fractional part becomes `<integer>`; a number with a fractional part becomes `<real>`.

Pass an optional `comments` key in `options`, mirroring the data structure, to inject `<!-- -->` XML comments before specific keys. Comments apply to the `xml` format only and are ignored for `binary` / `openstep`.

**Common uses:** generating Apple configuration profiles (`.mobileconfig`) for MDM deployment, WiFi/VPN payloads, app preference files, or anything else that downstream Apple tooling consumes.