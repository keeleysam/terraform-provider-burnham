package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type CIDRFilterVersionFunction struct{}

func NewCIDRFilterVersionFunction() function.Function { return &CIDRFilterVersionFunction{} }

func (f *CIDRFilterVersionFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_filter_version"
}

func (f *CIDRFilterVersionFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Filter a list of CIDRs by IP version",
		MarkdownDescription: "Returns only the CIDRs from `cidrs` that belong to the given IP `version` (4 or 6).",
		Parameters: []function.Parameter{
			function.ListParameter{
				Name:        "cidrs",
				Description: "The list of CIDRs to filter.",
				ElementType: types.StringType,
			},
			function.Int64Parameter{
				Name:        "version",
				Description: "The IP version to keep: 4 or 6.",
			},
		},
		Return: function.ListReturn{ElementType: types.StringType},
	}
}

func (f *CIDRFilterVersionFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var cidrs []string
	var version int64
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &cidrs, &version))
	if resp.Error != nil {
		return
	}

	result, err := iputil.FilterCIDRsByVersion(cidrs, version)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
