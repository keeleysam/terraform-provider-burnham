package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type IPInCIDRFunction struct{}

func NewIPInCIDRFunction() function.Function { return &IPInCIDRFunction{} }

func (f *IPInCIDRFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "ip_in_cidr"
}

func (f *IPInCIDRFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Check whether an IP address is within a CIDR",
		MarkdownDescription: "Returns `true` if `ip` is contained within `cidr`.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "ip", Description: "The IP address to check."},
			function.StringParameter{Name: "cidr", Description: "The CIDR to check against."},
		},
		Return: function.BoolReturn{},
	}
}

func (f *IPInCIDRFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var ip, cidr string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &ip, &cidr))
	if resp.Error != nil {
		return
	}

	result, err := iputil.IPInCIDR(ip, cidr)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
