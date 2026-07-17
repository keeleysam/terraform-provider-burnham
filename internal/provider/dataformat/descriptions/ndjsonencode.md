<!-- Edit here: this is the MarkdownDescription source for the burnham ndjsonencode function. docs/functions/ndjsonencode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Encodes a list as [NDJSON](https://github.com/ndjson/ndjson-spec): one JSON value per line, joined with `\n`, with a single trailing newline. Each element is JSON-encoded with sorted object keys and integer-rendering for whole-number values, matching the conventions of `jsonencode` here.

**Common uses:** generating fixture log files for tests, materializing event feeds, building bulk-import payloads.