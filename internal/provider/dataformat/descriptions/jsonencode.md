<!-- Edit here: this is the MarkdownDescription source for the burnham jsonencode function. docs/functions/jsonencode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Encodes a Terraform value as a pretty-printed JSON string with newlines and indentation. Unlike Terraform's built-in `jsonencode`, which produces a single compact line, this function returns output that's reviewable in pull requests and diff-friendly when written to a file.

The optional `options` object supports:

- `indent` (string): override the default tab indentation, e.g. `{ indent = "  " }` for two-space indent.
- `width` (number, default `0`): when greater than zero, pack an array of scalars onto a single line if it fits within this many columns. Objects are never packed, so every object stays alone on its line and you never see `}, {`, `[{` or `}]`. `0` keeps the default of one member or element per line. On a 372 KB policy, `width = 78` cuts 16,378 lines to 11,854 while remaining valid JSON.
- `escape_html` (bool, default `false`): when `false`, `<`, `>` and `&` are written literally, which is what you want for human-reviewed output. Terraform's built-in `jsonencode` (and Go's encoder) escape them to `\u003c` / `\u003e` / `\u0026`; set this to `true` to match that legacy behavior, e.g. when embedding JSON in an HTML `<script>` context.

Object keys are sorted alphabetically; whole numbers render without a decimal point.

Unknown keys are rejected. A misspelled option is an error rather than a silent no-op.

Note that `indent` is currently ignored when set to the empty string, which falls back to a tab. There is no compact/minified mode.

Unlike `hujsonencode`, which always emits trailing commas, this function always emits valid JSON. Terraform's plan renderer therefore gives it a compact structural diff rather than dumping the whole document, which matters a great deal on large files: the difference measured on a 372 KB policy is 0.25 s against 9 s, and 25 lines of plan output against 16,400. Prefer this function for anything large enough to review in a plan.

**Common uses:** rendering IAM policies, OpenAPI specs, or any structured JSON document that gets reviewed in PRs or written to disk via `local_file`.