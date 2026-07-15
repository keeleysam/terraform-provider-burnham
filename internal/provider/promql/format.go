// Package promql provides Terraform provider functions to validate and format PromQL (Prometheus Query Language) expressions.
//
// PromQL is hand-authored in alerting and recording rules (grafana_rule_group, Mimir rules, prometheus-operator PrometheusRule manifests) and dashboard panels. These functions catch invalid queries at plan time and canonicalize them.
//
// Backed by github.com/prometheus/prometheus/promql/parser, the official parser, so a query that validates here is valid in Prometheus.
package promql

import "github.com/prometheus/prometheus/promql/parser"

// parseExpr parses a PromQL expression with the stable (non-experimental) grammar.
func parseExpr(query string) (parser.Expr, error) {
	return parser.NewParser(parser.Options{}).ParseExpr(query)
}

// IsValid reports whether query is a valid PromQL expression. The Prometheus parser type-checks while parsing, so type errors (such as rate() on an instant vector) are caught too, not only syntax errors.
func IsValid(query string) bool {
	_, err := parseExpr(query)
	return err == nil
}

// Format parses query and returns its canonical PromQL serialization: normalized spacing and operator layout, with label matchers within a selector sorted alphabetically. When pretty is true it returns the parser's multi-line, indented form (only long sub-expressions are wrapped); otherwise a single canonical line. It errors on invalid input.
//
// The output is stable and idempotent. PromQL `#` comments are dropped (the parser discards them before building the AST).
func Format(query string, pretty bool) (string, error) {
	e, err := parseExpr(query)
	if err != nil {
		return "", err
	}
	if pretty {
		return parser.Prettify(e), nil
	}
	return e.String(), nil
}
