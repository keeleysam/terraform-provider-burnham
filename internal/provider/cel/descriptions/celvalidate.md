Returns `true` if `expr` is a syntactically valid [CEL](https://cel.dev) expression, `false` otherwise. Unlike `celformat`, it does not fail the plan on invalid input, so it is suitable for a boolean check (for example in a `precondition`).

Validation is syntax-only: it does not require variables or functions to be declared, and it does not check types or evaluate the expression.

By default it accepts the extensions real Kubernetes and GCP dialects use (optional types and two-variable comprehensions). Pass `{ strict = true }` to validate against base CEL instead, which rejects optional-navigation syntax (`?.`, `[?]`, `[?x]`, `{?k: v}`). This is useful for checking portability to a plain CEL host that has not enabled those extensions.

Backed by [cel-go](https://github.com/google/cel-go).