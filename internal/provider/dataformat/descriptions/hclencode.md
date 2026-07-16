Encodes a Terraform object as a sequence of HCL attribute statements (`key = value` lines), one per object member, in alphabetical key order.

Values render in their natural HCL form:

- Nested objects become HCL object literals (`{ ... }`).
- Lists become bracketed sequences.
- Primitives render as their natural HCL representation.

Output is formatted with `hclwrite.Format`, matching Terraform's canonical formatting.

~> **Note:** This is not the same as `provider::terraform::encode_tfvars`, which is `.tfvars`-specific. Use `hclencode` for emitting general-purpose HCL config files.