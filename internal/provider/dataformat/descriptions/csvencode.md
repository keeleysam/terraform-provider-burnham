<!-- Edit here: this is the MarkdownDescription source for the burnham csvencode function. docs/functions/csvencode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Encodes a list of objects as a CSV string. Each object becomes a row and object keys become columns. By default columns are sorted alphabetically and a header row is written.

Pass an optional `options` object:

- `columns` (list of strings): explicit column ordering. Only the listed columns are included.
- `no_header` (bool): omit the header row.

All cell values are converted to strings:

- numbers render as their string representation.
- bools render as `"true"` or `"false"`.
- nulls render as empty fields.

~> **Note:** Nested values (lists, objects) are not supported and fail the plan with an error.

Terraform has a built-in `csvdecode` for the reverse direction.

**Common uses:** generating CSV inputs for downstream loaders, exporting lookup tables, or producing reproducible spreadsheet-friendly output.