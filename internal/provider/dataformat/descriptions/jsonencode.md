<!-- Edit here: this is the MarkdownDescription source for the burnham jsonencode function. docs/functions/jsonencode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Encodes a Terraform value as a pretty-printed JSON string with newlines and indentation. Unlike Terraform's built-in `jsonencode`, which produces a single compact line, this function returns output that's reviewable in pull requests and diff-friendly when written to a file.

The optional `options` object supports:

- `indent` (string): override the default tab indentation, e.g. `{ indent = "  " }` for two-space indent.
- `escape_html` (bool, default `false`): when `false`, `<`, `>` and `&` are written literally, which is what you want for human-reviewed output. Terraform's built-in `jsonencode` (and Go's encoder) escape them to `\u003c` / `\u003e` / `\u0026`; set this to `true` to match that legacy behavior, e.g. when embedding JSON in an HTML `<script>` context.

Object keys are sorted alphabetically; whole numbers render without a decimal point.

**Common uses:** rendering IAM policies, OpenAPI specs, or any structured JSON document that gets reviewed in PRs or written to disk via `local_file`.