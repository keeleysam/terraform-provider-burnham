package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type CIDRVersionFunction struct{}

func NewCIDRVersionFunction() function.Function { return &CIDRVersionFunction{} }

func (f *CIDRVersionFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_version"
}

func (f *CIDRVersionFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Return the IP version of a CIDR",
		MarkdownDescription: "Returns `4` for IPv4 prefixes and `6` for IPv6 prefixes.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "cidr", Description: "The CIDR to inspect."},
		},
		Return: function.Int64Return{},
	}
}

func (f *CIDRVersionFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var cidr string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &cidr))
	if resp.Error != nil {
		return
	}

	result, err := iputil.CIDRVersion(cidr)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
