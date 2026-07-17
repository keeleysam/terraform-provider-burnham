<!-- Edit here: this is the MarkdownDescription source for the burnham hcldecode function. docs/functions/hcldecode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Parses an attribute-only [HCL2](https://github.com/hashicorp/hcl) document (a sequence of `key = value` attribute statements) and returns it as a Terraform object.

Values are evaluated as static literals only, because there is no eval context:

- Supported: numbers, strings, booleans, lists/tuples, and objects/maps.
- **Not** supported: references to variables, data sources, or function calls.

~> **Note:** Block syntax (`block_type "label" { ... }`) is not supported. Inputs containing top-level blocks are rejected with an error rather than silently dropped, so use `hcldecode` only for attribute-only documents.

For `.tfvars` files (which are themselves attribute-only HCL), `hcldecode` works fine. The built-in `provider::terraform::decode_tfvars` is an alternative tuned for that specific case.

**Common uses:** parsing simple HCL configs vendored alongside Terraform modules, reading attribute-only config files, or round-tripping HCL fragments emitted by other tooling.