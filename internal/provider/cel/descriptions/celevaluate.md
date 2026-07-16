Compiles and evaluates a [CEL](https://cel.dev) expression against variable bindings and returns the result as a Terraform value. Useful for testing the logic of an expression you built with `celencode`, and for computing or validating values inside a plan.

This evaluates **standard CEL only**: cel-go's standard library, its extension libraries (strings, math, lists, sets, encoders, bindings, two-variable comprehensions, regex, network), and optional types.

~> **Note:** Dialect-specific functions provided by a downstream host (GCP's `inIpRange`, Kubernetes' `quantity` / `authorizer`, and the like) are **not** available and will fail to compile, since this provider does not implement them.

Every variable referenced by the expression must be supplied in `vars`; an undeclared variable or function is a compile error. Variables are declared dynamically, so no type annotations are needed. Bindings and the format options below live in an optional second object argument: `celevaluate("x > 1", { vars = { x = 2 } })`.

-> **Note:** Evaluation is deterministic (CEL has no wall-clock or randomness), so results are stable across plan and apply.

Result values map to Terraform as follows (the timestamp, duration, and bytes renderings are overridable in options):

- A timestamp becomes an RFC 3339 string.
- A duration becomes a seconds string like `"5400s"`.
- Bytes become a base64 string.
- A map with non-string keys is an error (Terraform objects require string keys).
- An absent optional (`optional.none()`) becomes null, indistinguishable from a CEL null result.