<!-- Edit here: this is the MarkdownDescription source for the burnham json_patch function. docs/functions/json_patch.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Applies an [RFC 6902](https://www.rfc-editor.org/rfc/rfc6902) JSON Patch document to a Terraform value and returns the patched result. Use it for precise, ordered edits to a document, including element-level array changes and `test`-gated updates.

The patch is a list of operation objects. Each object has an `op`, a `path` (an [RFC 6901](https://www.rfc-editor.org/rfc/rfc6901) JSON Pointer), and operation-specific fields (`value`, `from`). The supported `op` values are:

- `"add"`
- `"remove"`
- `"replace"`
- `"move"`
- `"copy"`
- `"test"`

~> **Note:** Operations are applied in order. If any operation fails (including a failed `test`), the function returns an error and no partial state is produced.

Backed by [evanphx/json-patch](https://github.com/evanphx/json-patch).