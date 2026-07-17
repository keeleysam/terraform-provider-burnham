<!-- Edit here: this is the MarkdownDescription source for the burnham hclencode function. docs/functions/hclencode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Encodes a Terraform object as a sequence of HCL attribute statements (`key = value` lines), one per object member, in alphabetical key order.

Values render in their natural HCL form:

- Nested objects become HCL object literals (`{ ... }`).
- Lists become bracketed sequences.
- Primitives (strings, numbers, bools, null) render as HCL literals.

Output is formatted with `hclwrite.Format`, matching Terraform's canonical formatting.

~> **Note:** This is not the same as `provider::terraform::encode_tfvars`, which is `.tfvars`-specific. Use `hclencode` for emitting general-purpose HCL config files.