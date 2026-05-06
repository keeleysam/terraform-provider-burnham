package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type CIDRHostCountFunction struct{}

func NewCIDRHostCountFunction() function.Function { return &CIDRHostCountFunction{} }

func (f *CIDRHostCountFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_host_count"
}

func (f *CIDRHostCountFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Return the total number of addresses in a CIDR",
		MarkdownDescription: "Returns the total number of IP addresses in the CIDR, including the network " +
			"and broadcast addresses for IPv4. For very large IPv6 prefixes the result is capped at MaxInt64.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "cidr", Description: "The CIDR to count addresses in."},
		},
		Return: function.Int64Return{},
	}
}

func (f *CIDRHostCountFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var cidr string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &cidr))
	if resp.Error != nil {
		return
	}

	result, err := iputil.CIDRHostCount(cidr)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
