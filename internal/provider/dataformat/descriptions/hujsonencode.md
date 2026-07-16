Encodes a Terraform value as a HuJSON string with trailing commas and pretty-printed formatting. By default every object member and array element gets its own line, producing diff-friendly output.

Pass an optional `options` object with these keys:

- `indent` (string): override the default tab indentation.
- `compact` (bool): opt in to hujson.Format's "fit on one line if it can" packing instead of the default always-expanded layout.
- `escape_html` (bool, default `false`): when `false`, `<`, `>` and `&` are written literally, matching `jsonencode`. Set to `true` to escape them to `\u003c` / `\u003e` / `\u0026` (the form Go's encoder produces), e.g. when embedding output in an HTML `<script>` context.
- `comments` (object): mirror the data structure with string values that become comments placed before the matching key. Single-line strings render as `//` comments; strings containing `\n` render as `/* */` block comments. Array elements are addressed by stringified index (`"0"`, `"1"`, …).

**Common uses:** generating Tailscale ACL files, writing human-editable config snapshots, or producing JSON-like documents where reviewers benefit from inline annotations.