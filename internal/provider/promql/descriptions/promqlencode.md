Builds a canonical [PromQL](https://prometheus.io/docs/prometheus/latest/querying/basics/) query from a structured HCL value, so you can assemble a query from Terraform data with correct quoting and no fragile string interpolation.

The pain it removes is selectors. A label value or regex built with `${...}` breaks on quotes or special characters, whereas here the matcher values are quoted correctly for you.

The tree is modeled on the Prometheus AST, using the node types the experimental `/api/v1/parse_query` endpoint exposes. Each construct is a single-key object naming a node type (not `parse_query`'s literal wire format).

### Leaves

- A bare number is a numeric literal.
- A bare string is a string literal.

### Selectors

- `vectorSelector` takes the optional `name` and `matchers` (supply at least one to form a valid selector, for example just `name = "up"` or just `matchers = [{ name = "job", type = "=", value = "api" }]`), plus the optional `offset` and `at`. `at` is a Unix timestamp in seconds (the `@` modifier), or the string `"start"` or `"end"`. Each matcher is `{ name, type, value }`, where `type` is `=`, `!=`, `=~`, or `!~`.
- `matrixSelector` is a range vector: the same fields plus a required `range` (for example `"5m"`).

```hcl
{ vectorSelector = { name = "http_requests_total", matchers = [{ name = "job", type = "=", value = "api" }], offset = "5m", at = 1609746000 } }
```

### Calls and aggregations

- `call` takes `func` (a PromQL function name) and `args`, for example `{ call = { func = "rate", args = [...] } }`. Functions Prometheus flags as experimental are rejected.
- `aggregation` takes `op`, `expr`, an optional grouping (`by` or `without`), and an optional `param`. `op` is one of `sum`, `avg`, `min`, `max`, `count`, `count_values`, `quantile`, `stddev`, `stdvar`, `topk`, `bottomk`, or `group`. `param` supplies the extra argument for `topk`, `bottomk`, `quantile`, and `count_values`.

### Operators

- `binaryExpr` takes `op`, `lhs`, and `rhs`. `op` is an arithmetic, comparison, or set operator. Optional `bool` forces a boolean result, and `on`/`ignoring` with `group_left`/`group_right` control vector matching, for example `{ binaryExpr = { op = "/", lhs = ..., rhs = ..., bool = true, on = ["job"], group_left = ["instance"] } }`.
- `paren` wraps a sub-expression in parentheses.
- `neg` applies unary `-`, and `pos` applies unary `+`.

### Subqueries

- `subquery` takes `expr` and `range`, plus the optional `step`, `offset`, and `at`, for example `{ subquery = { expr = ..., range = "30m", step = "1m", offset = "5m", at = ... } }`.

### Escape hatch

- `raw` embeds a hand-written PromQL fragment: `{ raw = "<promql>" }`. The fragment is parsed, and so validated, before use.

-> **Note:** The parser's own AST is built and re-serialized, so the output is canonical (byte-identical to `promqlformat`) and `promqlencode` never emits an invalid query. Backed by [prometheus/prometheus](https://github.com/prometheus/prometheus)'s own parser.