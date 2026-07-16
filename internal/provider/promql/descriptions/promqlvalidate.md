Returns `true` if `query` is a valid [PromQL](https://prometheus.io/docs/prometheus/latest/querying/basics/) expression, `false` otherwise.

Unlike `promqlformat`, it does not fail the plan on invalid input, so it suits a boolean check in a `precondition` guarding a hand-written query (in a `grafana_rule_group`, a Mimir rule, a `PrometheusRule` manifest, or a dashboard panel).

The Prometheus parser type-checks while parsing, so this catches type errors too, not only syntax. Each of these returns `false`:

- `rate()` on an instant vector.
- A range vector used in arithmetic.
- A string where a scalar is required.

-> **Note:** Backed by [prometheus/prometheus](https://github.com/prometheus/prometheus)'s own parser, so a query that validates here is valid in Prometheus. A query longer than 64 KiB is reported as `false` without parsing, keeping the function from ever failing the plan.