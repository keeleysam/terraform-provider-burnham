Builds a single [Cedar](https://www.cedarpolicy.com) policy, in its human-readable text form, from a structured HCL value, so you can assemble a policy from Terraform data with no string templating. The result is a canonical policy statement suitable for `aws_verifiedpermissions_policy`.

The HCL you pass mirrors Cedar's own JSON policy format, the EST, one-to-one, so `cedardecode` produces the same shape. Cedar defines the EST directly, and this function walks that structured value (not a JSON string).

### Top-level object

- `effect`: `"permit"` or `"forbid"`.
- `principal`, `action`, `resource`: scope objects (see below).
- `conditions`: an optional list of clauses, each an object `{ kind = "when" or "unless", body = <Cedar EST expression tree> }`.

### Scope objects

Each scope is an object with an `op` field. The shape of the rest of the object depends on `op`:

- `"=="`: `{ op = "==", entity = { type = ..., id = ... } }`.
- `"in"`: `{ op = "in", entity = { type = ..., id = ... } }`, or a set with `entities = [ { type = ..., id = ... }, ... ]`.
- `"is"`: `{ op = "is", entity_type = "Namespace::Type" }`, with an optional `in = { entity = { type = ..., id = ... } }`. Note this uses `entity_type` (a string), not `entity`.
- `"All"`: an unconstrained scope, for example a bare `resource`. It carries no entity.

-> **Note:** The simplest way to get the EST shape for a non-trivial policy or condition is to write it as Cedar text and run it through `cedardecode`.

The tree is validated as it is converted, so `cedarencode` never emits a syntactically invalid policy, and the output is canonical (byte-identical to what `cedarformat` produces for one policy). Backed by [cedar-go](https://github.com/cedar-policy/cedar-go), the official Go implementation of Cedar.