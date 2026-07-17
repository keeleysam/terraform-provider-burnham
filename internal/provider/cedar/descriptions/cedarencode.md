<!-- Edit here: this is the MarkdownDescription source for the burnham cedarencode function. docs/functions/cedarencode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Builds a single [Cedar](https://www.cedarpolicy.com) policy, in its human-readable text form, from a structured HCL value, so you can assemble a policy from Terraform data with no string templating. The result is a canonical policy statement suitable for `aws_verifiedpermissions_policy`.

The HCL you pass mirrors Cedar's own JSON policy format, the EST, one-to-one, so `cedardecode` produces the same shape. Cedar defines the EST directly, and this function walks that structured value (not a JSON string).

### Top-level object

- `effect`: `"permit"` or `"forbid"`.
- `principal`, `action`, `resource`: scope objects (see below).
- `conditions`: an optional list of clauses, each an object `{ kind = "when" or "unless", body = <Cedar EST expression tree> }`.

### Scope objects

Each scope is an object with an `op` field. The shape of the rest of the object depends on `op`:

- `"=="`: `{ op = "==", entity = { type = ..., id = ... } }`.
- `"in"`: `{ op = "in", entity = { type = ..., id = ... } }`. The set form with `entities = [ { type = ..., id = ... }, ... ]` is valid only for `action` (Cedar's `action in [Action::"a", Action::"b"]`); `principal` and `resource` accept a single `entity` only.
- `"is"`: `{ op = "is", entity_type = "Namespace::Type" }`, with an optional `in = { entity = { type = ..., id = ... } }`. Note this uses `entity_type` (a string), not `entity`. Valid only for `principal` and `resource`, not `action`.
- `"All"`: an unconstrained scope, for example a bare `resource`. It carries no entity.

-> **Note:** The simplest way to get the EST shape for a non-trivial policy or condition is to write it as Cedar text and run it through `cedardecode`.

The tree is validated as it is converted, so `cedarencode` never emits a syntactically invalid policy, and the output is canonical and deterministic: record-literal keys are always emitted in lexicographic order. It matches what `cedarformat` produces for one policy, except when the source policy lists record-literal keys in some other order (`cedarformat` preserves the source order, `cedarencode` sorts them). Backed by [cedar-go](https://github.com/cedar-policy/cedar-go), the official Go implementation of Cedar.