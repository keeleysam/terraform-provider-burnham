Returns `true` if `expression` is a syntactically valid [JSONata](https://jsonata.org/) expression, `false` otherwise. Unlike `jsonata_query`, it does not fail the plan on invalid input, so it is suitable for a boolean check (for example in a `precondition`).

Validation is syntax-only: it does not require the referenced fields to exist, and it does not evaluate the expression.

-> An expression that references the non-deterministic builtins (`$now`, `$millis`, `$random`) is still syntactically valid, so this returns `true`. Those builtins are rejected only when `jsonata_query` evaluates an expression, not when its syntax is checked.

~> **Note:** An expression longer than the supported maximum returns `false` rather than an error, so this function never fails the plan.

Backed by [recolabs/gnata](https://github.com/recolabs/gnata), a pure-Go implementation of JSONata 2.x.