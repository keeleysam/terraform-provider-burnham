package promql

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = (*PromQLDecodeFunction)(nil)

type PromQLDecodeFunction struct{}

func NewPromQLDecodeFunction() function.Function { return &PromQLDecodeFunction{} }

func (f *PromQLDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "promqldecode"
}

func (f *PromQLDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Parse a PromQL query into a promqlencode data tree",
		MarkdownDescription: "Parses a [PromQL](https://prometheus.io/docs/prometheus/latest/querying/basics/) query and returns it as the HCL data tree that `promqlencode` consumes, so `provider::burnham::promqlencode(provider::burnham::promqldecode(query))` round-trips to the canonical form of `query`. It is the tool for lifting a hand-written query into the structured model, whether to edit part of it programmatically or to check what tree a query corresponds to.\n\nThe returned tree uses the same node vocabulary as `promqlencode`: each construct is a single-key object naming a node type (`vectorSelector`, `matrixSelector`, `call`, `aggregation`, `binaryExpr`, `subquery`, `paren`, `neg`, `pos`), a bare number is a numeric literal, and a bare string is a string literal. Every node is fully structured, so the tree never contains a `raw` fragment. The implicit `__name__` matcher that a bare metric name carries is folded into the `name` field rather than repeated as a matcher.\n\nBecause the parser normalizes as it reads (label matchers sort alphabetically, redundant braces drop, spacing is regularized), the tree reflects the canonical query rather than the original byte layout, and re-encoding it yields `promqlformat`'s output. It fails the plan on invalid input, the same as `promqlformat`. Backed by [prometheus/prometheus](https://github.com/prometheus/prometheus)'s own parser.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "query",
				Description: "A PromQL expression to parse into a data tree.",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *PromQLDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var query string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &query))
	if resp.Error != nil {
		return
	}
	if len(query) > promqlMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("query exceeds maximum supported length of %d bytes", promqlMaxInputBytes))
		return
	}

	tree, err := Decode(query)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}
	value, err := nodeToAttr(tree)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(value)))
}
