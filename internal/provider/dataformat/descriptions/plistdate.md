<!-- Edit here: this is the MarkdownDescription source for the burnham plistdate function. docs/functions/plistdate.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns a tagged object representing an [`NSDate`](https://developer.apple.com/documentation/foundation/nsdate) plist value. When passed to `plistencode`, this produces a `<date>` XML element with the given timestamp. `plistdecode` returns the same tagged-object shape for `<date>` elements, so encode/decode round-trips preserve the type.

The input must be an [RFC 3339](https://www.rfc-editor.org/rfc/rfc3339) timestamp string (e.g. `"2026-06-01T00:00:00Z"`).

**Common uses:** setting `RemovalDate`, `ExpirationDate`, or other date-typed fields in Apple configuration profiles.