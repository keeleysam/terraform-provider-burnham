<!-- Edit here: this is the MarkdownDescription source for the burnham kdlencode function. docs/functions/kdlencode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Encodes a Terraform list of node objects as a [KDL document](https://kdl.dev/) string.

Each node object should have these keys:

- `name` (string)
- `args` (list)
- `props` (map)
- `children` (list of child nodes)

Output is KDL v2 by default. Pass `options = { version = "v1" }` for the legacy KDL v1 grammar.

**Common uses:** generating KDL configuration for tools that have adopted the format, or producing structured-document artifacts where KDL's terse syntax is more readable than YAML or JSON.