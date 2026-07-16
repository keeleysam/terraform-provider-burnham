package network

import (
	"context"
	_ "embed"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type CIDRMergeFunction struct{}

func NewCIDRMergeFunction() function.Function { return &CIDRMergeFunction{} }

func (f *CIDRMergeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_merge"
}

//go:embed descriptions/cidr_merge.md
var cidrMergeDescription string

func (f *CIDRMergeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Aggregate CIDRs into the smallest possible list",
		MarkdownDescription: cidrMergeDescription,
		Parameters: []function.Parameter{
			function.ListParameter{
				Name:        "cidrs",
				Description: "List of CIDR strings to merge/aggregate.",
				ElementType: types.StringType,
			},
		},
		Return: function.ListReturn{ElementType: types.StringType},
	}
}

func (f *CIDRMergeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var cidrsArg types.List
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &cidrsArg))
	if resp.Error != nil {
		return
	}
	cidrs, argErr := cidrListArg(cidrsArg, 0, "cidrs")
	if argErr != nil {
		resp.Error = argErr
		return
	}

	result, err := iputil.MergeCIDRs(cidrs)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
