<!-- Edit here: this is the MarkdownDescription source for the burnham celdecode function. docs/functions/celdecode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Parses a [CEL](https://cel.dev) expression string and returns it as the HCL data tree that `celencode` consumes, so `provider::burnham::celencode(provider::burnham::celdecode(expr))` round-trips to the canonical form of `expr`. Primarily a tool for testing and for migrating hand-written CEL into the data model.

The optional second argument selects the notation returned:

- `canonical`: the verbose `cel/expr/syntax.proto` field-name form (`call_expr`, `ident_expr`, `select_expr`, `const_expr`, `list_expr`, `struct_expr`); operators are calls with the canonical function names.
- `standard` (default): the readable form: type-name keys (`ident`, `call`, ...), folded `ident` reference paths, bare literals, and CEL operator tokens (`"=="`, `"&&"`, `"in"`). Nested `&&`/`||` are flattened into a single variadic list.
- `aliased`: like `standard` but with the friendly word aliases (`and`, `or`, `not`, `eq`, `ne`, `lt`, ...).

All three re-encode through `celencode` to the same CEL string. Validation is syntax-only (via cel-go with optional types and two-variable comprehensions enabled), so any syntactically valid CEL decodes. The return is a dynamic value; CEL list literals decode to Terraform tuples (heterogeneous), which `celencode` accepts on the way back.