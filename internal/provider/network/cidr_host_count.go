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
		MarkdownDescription: "Returns the total number of IP addresses in the CIDR, including the network and broadcast addresses for IPv4. For very large IPv6 prefixes the result is capped at MaxInt64.",
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

type CIDRUsableHostCountFunction struct{}

func NewCIDRUsableHostCountFunction() function.Function { return &CIDRUsableHostCountFunction{} }

func (f *CIDRUsableHostCountFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_usable_host_count"
}

func (f *CIDRUsableHostCountFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Return the number of usable host addresses in a CIDR",
		MarkdownDescription: "Returns the number of usable host addresses in the CIDR. For IPv4, the network and broadcast addresses are subtracted, with special cases: `/31` returns 2 (point-to-point, RFC 3021), `/32` returns 1 (host route). For IPv6, all addresses are considered usable.\n\n**Common uses:** asserting a subnet is large enough for a given number of workloads without manually subtracting 2 everywhere; sizing auto-scaling groups or node pools based on the actual available IP space in the target subnet.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "cidr", Description: "The CIDR to count usable hosts in."},
		},
		Return: function.Int64Return{},
	}
}

func (f *CIDRUsableHostCountFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var cidr string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &cidr))
	if resp.Error != nil {
		return
	}

	result, err := iputil.CIDRUsableHostCount(cidr)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
