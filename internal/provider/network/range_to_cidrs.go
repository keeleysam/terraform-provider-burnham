package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type RangeToCIDRsFunction struct{}

func NewRangeToCIDRsFunction() function.Function { return &RangeToCIDRsFunction{} }

func (f *RangeToCIDRsFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "range_to_cidrs"
}

func (f *RangeToCIDRsFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Convert an IP range to a minimal list of CIDRs",
		MarkdownDescription: "Converts an inclusive IP range `[first_ip, last_ip]` into the minimal list of " +
			"CIDRs that exactly covers the range. Both IPs must be the same address family (IPv4 or IPv6).",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "first_ip",
				Description: "The first (lowest) IP address in the range.",
			},
			function.StringParameter{
				Name:        "last_ip",
				Description: "The last (highest) IP address in the range.",
			},
		},
		Return: function.ListReturn{ElementType: types.StringType},
	}
}

func (f *RangeToCIDRsFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var firstIP, lastIP string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &firstIP, &lastIP))
	if resp.Error != nil {
		return
	}

	result, err := iputil.RangeToCIDRs(firstIP, lastIP)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
