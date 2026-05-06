package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type CIDRSubtractFunction struct{}

func NewCIDRSubtractFunction() function.Function { return &CIDRSubtractFunction{} }

func (f *CIDRSubtractFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_subtract"
}

func (f *CIDRSubtractFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Subtract CIDRs from an input list",
		MarkdownDescription: "Returns the set of CIDRs that remain after removing all addresses covered by " +
			"the `exclude` list from the `input` list. The result is merged into the smallest equivalent set.",
		Parameters: []function.Parameter{
			function.ListParameter{
				Name:        "input",
				Description: "The base list of CIDRs to subtract from.",
				ElementType: types.StringType,
			},
			function.ListParameter{
				Name:        "exclude",
				Description: "The list of CIDRs to remove from the input.",
				ElementType: types.StringType,
			},
		},
		Return: function.ListReturn{ElementType: types.StringType},
	}
}

func (f *CIDRSubtractFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input, exclude []string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input, &exclude))
	if resp.Error != nil {
		return
	}

	result, err := iputil.SubtractCIDRs(input, exclude)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
