Parses a hand-written [CEL](https://cel.dev) expression string and returns its canonical form, failing the plan with a diagnostic if the expression is not syntactically valid. Use `celvalidate` instead if you want a boolean rather than a hard failure.

The returned string is normalized (canonical quoting, spacing, and precedence-minimal parentheses) and stable across runs.

Parsing is syntax-only and dialect-neutral: it does not require variables or functions to be declared, so it never rejects a valid expression that uses environment-specific functions or variables, and standard macros (`has`, `all`, `exists`, `exists_one`, `map`, `filter`) keep their sugar. It accepts cel-go with optional types and two-variable comprehensions enabled, so it is not strictly base-CEL grammar.

Pass a `format` options object to pretty-print or wrap the output (see the options argument).

Backed by [cel-go](https://github.com/google/cel-go) and [celfmt](https://github.com/elastic/celfmt).