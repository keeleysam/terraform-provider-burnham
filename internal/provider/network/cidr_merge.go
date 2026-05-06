package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type CIDRMergeFunction struct{}

func NewCIDRMergeFunction() function.Function { return &CIDRMergeFunction{} }

func (f *CIDRMergeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_merge"
}

func (f *CIDRMergeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Aggregate CIDRs into the smallest possible list",
		MarkdownDescription: "Merges a list of CIDR strings into the smallest equivalent set by removing " +
			"redundant prefixes and combining sibling pairs into supernets. Supports both IPv4 and IPv6.",
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
	var cidrs []string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &cidrs))
	if resp.Error != nil {
		return
	}

	result, err := iputil.MergeCIDRs(cidrs)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
