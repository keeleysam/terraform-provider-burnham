package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type CIDRWildcardFunction struct{}

func NewCIDRWildcardFunction() function.Function { return &CIDRWildcardFunction{} }

func (f *CIDRWildcardFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_wildcard"
}

func (f *CIDRWildcardFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Return the wildcard mask (inverse of subnet mask) for an IPv4 CIDR",
		MarkdownDescription: "Returns the wildcard mask for the given IPv4 CIDR. For example, `10.0.0.0/24` → `0.0.0.255`. IPv6 CIDRs return an error.\n\n" +
			"**Common uses:** generating Cisco IOS ACL entries, AWS network ACL wildcard fields, " +
			"firewall rules that use wildcard mask notation instead of prefix length.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "cidr", Description: "An IPv4 CIDR."},
		},
		Return: function.StringReturn{},
	}
}

func (f *CIDRWildcardFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var cidr string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &cidr))
	if resp.Error != nil {
		return
	}

	result, err := iputil.CIDRWildcard(cidr)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
