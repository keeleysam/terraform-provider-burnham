package promql

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

var _ function.Function = (*PromQLValidateFunction)(nil)

type PromQLValidateFunction struct{}

func NewPromQLValidateFunction() function.Function { return &PromQLValidateFunction{} }

func (f *PromQLValidateFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "promqlvalidate"
}

func (f *PromQLValidateFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Report whether a string is a valid PromQL expression",
		MarkdownDescription: "Returns `true` if `query` is a valid [PromQL](https://prometheus.io/docs/prometheus/latest/querying/basics/) expression, `false` otherwise. Unlike `promqlformat`, it does not fail the plan on invalid input, so it suits a boolean check in a `precondition` guarding a hand-written query (in a `grafana_rule_group`, a Mimir rule, a `PrometheusRule` manifest, or a dashboard panel).\n\nThe Prometheus parser type-checks while parsing, so this catches type errors too, not only syntax: `rate()` on an instant vector, a range vector in arithmetic, or a string where a scalar is required all return `false`. Backed by [prometheus/prometheus](https://github.com/prometheus/prometheus)'s own parser, so a query that validates here is valid in Prometheus.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "query",
				Description: "A PromQL expression to check.",
			},
		},
		Return: function.BoolReturn{},
	}
}

func (f *PromQLValidateFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var query string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &query))
	if resp.Error != nil {
		return
	}
	if len(query) > promqlMaxInputBytes {
		// Over the size guard, report not-valid rather than failing the plan, keeping the "does not fail the plan" contract absolute (a query this large is not a real one).
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, false))
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, IsValid(query)))
}
