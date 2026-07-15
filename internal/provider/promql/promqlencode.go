package promql

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = (*PromQLEncodeFunction)(nil)

type PromQLEncodeFunction struct{}

func NewPromQLEncodeFunction() function.Function { return &PromQLEncodeFunction{} }

func (f *PromQLEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "promqlencode"
}

func (f *PromQLEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Build a PromQL query from an HCL data tree",
		MarkdownDescription: "Builds a canonical [PromQL](https://prometheus.io/docs/prometheus/latest/querying/basics/) query from a structured HCL value, so you assemble a query from Terraform data with correct quoting and no fragile string interpolation. The pain it removes is selectors: a label value or regex built with `${...}` breaks on quotes or special characters, whereas here the matcher values are quoted correctly for you.\n\nThe tree is modeled on the Prometheus AST, using the node types the experimental `/api/v1/parse_query` endpoint exposes. Each construct is a single-key object naming a node type (not `parse_query`'s literal wire format); a bare number is a numeric literal and a bare string is a string literal:\n\n- `{ vectorSelector = { name = \"http_requests_total\", matchers = [ { name = \"job\", type = \"=\", value = \"api\" } ], offset = \"5m\", at = 1609746000 } }` (matcher `type` is `=`, `!=`, `=~`, or `!~`; `offset`/`at` optional).\n- `{ matrixSelector = { name = ..., matchers = [...], range = \"5m\" } }` (a range vector).\n- `{ call = { func = \"rate\", args = [ ... ] } }`.\n- `{ aggregation = { op = \"sum\", by = [\"job\"], expr = ... } }` (`op` is `sum`/`avg`/`min`/`max`/`count`/`count_values`/`quantile`/`stddev`/`stdvar`/`topk`/`bottomk`/`group`; `by` or `without`; `param` for topk/quantile/count_values).\n- `{ binaryExpr = { op = \"/\", lhs = ..., rhs = ..., bool = true, on = [\"job\"], group_left = [\"instance\"] } }` (`op` is an arithmetic, comparison, or set operator; optional `bool`, and `on`/`ignoring` with `group_left`/`group_right` for vector matching).\n- `{ subquery = { expr = ..., range = \"30m\", step = \"1m\" } }`, `{ paren = ... }`, `{ neg = ... }`.\n- `{ raw = \"<promql>\" }` embeds a hand-written fragment (parsed, and so validated).\n\nThe parser's own AST is built and re-serialized, so the output is canonical (byte-identical to `promqlformat`) and `promqlencode` never emits an invalid query. Backed by [prometheus/prometheus](https://github.com/prometheus/prometheus)'s own parser.",
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "query",
				Description: "The query as a data tree mirroring the PromQL AST.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *PromQLEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var query types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &query))
	if resp.Error != nil {
		return
	}
	if hasUnknown(query) {
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.StringUnknown()))
		return
	}

	tree, err := terraformToNode(query.UnderlyingValue())
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to read query: "+err.Error())
		return
	}
	out, err := Encode(tree)
	if errors.Is(err, errInvalidOutput) {
		resp.Error = function.NewFuncError(err.Error())
		return
	}
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, out))
}
