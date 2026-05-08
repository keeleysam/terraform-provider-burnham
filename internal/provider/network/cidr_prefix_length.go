package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type CIDRPrefixLengthFunction struct{}

func NewCIDRPrefixLengthFunction() function.Function { return &CIDRPrefixLengthFunction{} }

func (f *CIDRPrefixLengthFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_prefix_length"
}

func (f *CIDRPrefixLengthFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Return the prefix length (/N) of a CIDR as a number",
		MarkdownDescription: "Extracts and returns just the prefix length from a CIDR string.\n\n**Common uses:** passing prefix lengths to BGP route-map configurations, conditional logic based on subnet size, feeding into `cidrsubnet` calls.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "cidr", Description: "The CIDR to inspect."},
		},
		Return: function.Int64Return{},
	}
}

func (f *CIDRPrefixLengthFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var cidr string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &cidr))
	if resp.Error != nil {
		return
	}

	result, err := iputil.CIDRPrefixLength(cidr)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
