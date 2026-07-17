<!-- Edit here: this is the MarkdownDescription source for the burnham plistdata function. docs/functions/plistdata.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns a tagged object representing an [`NSData`](https://developer.apple.com/documentation/foundation/nsdata) plist value. When passed to `plistencode`, this produces a `<data>` XML element with the given binary payload. `plistdecode` returns the same tagged-object shape for `<data>` elements, so encode/decode round-trips preserve the type.

The input must be a base64-encoded string. Pair with `filebase64("path/to/file")` to embed file contents like certificates, profile-signing material, or images.

**Common uses:** embedding signing certificates, custom icons, or other binary blobs into Apple configuration profiles.