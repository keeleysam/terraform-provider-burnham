package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type CIDRExpandFunction struct{}

func NewCIDRExpandFunction() function.Function { return &CIDRExpandFunction{} }

func (f *CIDRExpandFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_expand"
}

func (f *CIDRExpandFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Expand a CIDR into a list of individual IP addresses",
		MarkdownDescription: "Returns every individual IP address within the given CIDR as a list of strings. " +
			"Returns an error if the CIDR contains more than 65536 addresses to prevent accidental large expansions.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "cidr",
				Description: "The CIDR to expand.",
			},
		},
		Return: function.ListReturn{ElementType: types.StringType},
	}
}

func (f *CIDRExpandFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var cidr string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &cidr))
	if resp.Error != nil {
		return
	}

	result, err := iputil.ExpandCIDR(cidr)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
