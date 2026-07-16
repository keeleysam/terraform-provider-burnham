Builds a [CEL](https://cel.dev) (Common Expression Language) expression string from a structured HCL value, so you can assemble expressions from Terraform data (variables, `for` expressions, `merge`, `concat`) with no string templating. The result is a canonical, deterministic CEL string suitable for GCP IAM / Access Context Manager conditions, Workload Identity Federation, Cloud Armor, Kubernetes CEL, and any other CEL sink.

The input follows the CEL canonical AST (`cel/expr/syntax.proto`), using its node and field names rather than an invented vocabulary. Two notations are accepted and may be freely mixed: a readable **surface** notation and the verbose **canonical** notation.

Bare integral numbers become a CEL `int` and bare non-integral numbers become a CEL `double`. Use `{ const = { double_value = ... } }` to force an integral value (like `1.0`) to a `double`, and `{ const = { uint64_value = ... } }` for an unsigned or large (> 2^63-1) value.

In the surface (readable) notation you mark only references; everything else is a bare literal.

### Leaves

- Bare strings, numbers, bools, and `null` are literals, and a bare list is a list literal.
- A reference (variable, field path, or enum) is the only marked leaf: `{ ident = "device.os_type" }`. Dotted and `['key']` index paths expand automatically.

### Operators

Write an operator as a single-key object mapping a CEL surface token or friendly alias to its operands, for example `{ "==" = [a, b] }` or `{ eq = [a, b] }`. Supported aliases:

- Comparison: `eq`, `ne`, `lt`, `le`, `gt`, `ge`
- Logical: `and`, `or`, `not`
- Membership and conditional: `in`, `cond`
- Arithmetic: `add`, `sub`, `mul`, `div`, `mod`, `neg`
- Indexing: `index`

### Calls and macros

- Calls: `{ call = { function = "startsWith", target = { ident = "resource.name" }, args = ["prod-"] } }`.
- Macros are calls whose function is `has`, `all`, `exists`, `exists_one`, `map`, or `filter`, with the bound variable passed as an `{ ident = "g" }` argument (`has` takes a single field-selection argument). Author comprehension macros in this call form, not as a raw `comprehension_expr`.

### Explicit literals and structs

- `{ const = ["US", "CA"] }` forces a literal. This is recursive: lists, maps, and typed constants like `{ const = { double_value = 1 } }`.
- `{ struct = { message_name = "T", fields = { f = 1 } } }` constructs a message.

~> **Note:** A single-key `const` map whose key is a CEL constant kind, e.g. `{ const = { int64_value = 5 } }`, is read as that typed scalar, not as a one-entry map. Spell such a map via `struct_expr` map entries or `raw`.

### Escape hatch

- `{ raw = "a.b.exists(x, x > 0)" }` embeds hand-written CEL.

### Canonical notation

Instead of the readable keys, use the `syntax.proto` field names directly, where operators are calls with the canonical function names:

```hcl
{ call_expr = { function = "_==_", args = [
  { select_expr = { operand = { ident_expr = { name = "device" } }, field = "os_type" } },
  { ident_expr = { name = "OsType" } },
] } }
```

The canonical keys are `ident_expr`, `select_expr`, `call_expr`, `const_expr`, `list_expr`, and `struct_expr`.

### Optional types and two-variable comprehensions

These cover the extensions the Kubernetes and GCP dialects use.

- Optional navigation goes in an `ident` (or `raw`) path: `{ ident = "msg.?field" }` and `{ ident = "m[?k]" }`.
- An optional list element uses `{ optional = <expr> }` inside a list (CEL `[?x]`).
- An optional map or struct entry sets `optional_entry = true` on a `struct_expr` entry (CEL `{?k: v}`).
- `optional.of`, `orValue`, `hasValue`, `value`, and `optMap` are ordinary calls.
- Two-variable comprehensions (`m.all(k, v, ...)`, `transformList`, `transformMap`, `transformMapEntry`) are calls with two bound-variable `ident` arguments.

-> **Note:** `ident` accepts only reference paths (identifier, field, index, optional navigation). A full expression must use `raw`.

~> **Note:** The output is always validated (syntax only, dialect-neutral) before it is returned, so `celencode` can never produce a syntactically invalid CEL string.

Backed by [cel-go](https://github.com/google/cel-go), which handles operator precedence, parenthesization, and canonical quoting. The normalized output is stable across runs, so it does not churn the plan.

Pass a `format` options object to pretty-print or wrap the output (see the options argument).