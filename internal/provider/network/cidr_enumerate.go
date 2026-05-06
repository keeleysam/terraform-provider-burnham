package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type CIDREnumerateFunction struct{}

func NewCIDREnumerateFunction() function.Function { return &CIDREnumerateFunction{} }

func (f *CIDREnumerateFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_enumerate"
}

func (f *CIDREnumerateFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Enumerate all sub-CIDRs of a given size within a CIDR",
		MarkdownDescription: "Returns every possible subnet of size `(parent prefix length + newbits)` within `cidr`. " +
			"For example, `cidr_enumerate(\"10.0.0.0/24\", 2)` returns all four /26 subnets within the /24. " +
			"Returns an error if the result would exceed 65536 subnets.\n\n" +
			"**Common uses:** pre-computing all AZ-sized subnets within a region block for later `element()` " +
			"selection; generating candidate subnets to feed into an IP address management workflow.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "cidr",
				Description: "The parent CIDR to subdivide.",
			},
			function.Int64Parameter{
				Name:        "newbits",
				Description: "Number of additional prefix bits. Each +1 halves the subnet size.",
			},
		},
		Return: function.ListReturn{ElementType: types.StringType},
	}
}

func (f *CIDREnumerateFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var cidr string
	var newbits int64
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &cidr, &newbits))
	if resp.Error != nil {
		return
	}

	result, err := iputil.EnumerateCIDR(cidr, newbits)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
