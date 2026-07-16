Parses a [PromQL](https://prometheus.io/docs/prometheus/latest/querying/basics/) query and returns it as the HCL data tree that `promqlencode` consumes, so `promqlencode(promqldecode(query))` round-trips to the canonical form of `query`.

Use it to lift a hand-written query into the structured model, whether to edit part of it programmatically or to check what tree a query corresponds to.

The returned tree uses the same node vocabulary as `promqlencode`. Each construct is a single-key object naming a node type (`vectorSelector`, `matrixSelector`, `call`, `aggregation`, `binaryExpr`, `subquery`, `paren`, `neg`, `pos`), a bare number is a numeric literal, and a bare string is a string literal.

Two normalizations to expect in the output:

- Every node is fully structured, so the tree never contains a `raw` fragment.
- The implicit `__name__` matcher that a bare metric name carries is folded into the `name` field rather than repeated as a matcher.

Because the parser normalizes as it reads (label matchers sort alphabetically, redundant braces drop, spacing is regularized), the tree reflects the canonical query rather than the original byte layout, and re-encoding it yields `promqlformat`'s output.

~> **Note:** Fails the plan on invalid input, or on a query longer than 64 KiB, the same as `promqlformat`. Backed by [prometheus/prometheus](https://github.com/prometheus/prometheus)'s own parser.