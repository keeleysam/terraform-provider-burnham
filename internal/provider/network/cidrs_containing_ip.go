package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type CIDRsContainingIPFunction struct{}

func NewCIDRsContainingIPFunction() function.Function { return &CIDRsContainingIPFunction{} }

func (f *CIDRsContainingIPFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidrs_containing_ip"
}

func (f *CIDRsContainingIPFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Return every CIDR from a list that contains the given IP address",
		MarkdownDescription: "Returns every CIDR from `cidrs` that contains `ip` as a list. Returns an empty list if none match. Multiple CIDRs may match when the list contains overlapping prefixes (e.g. a summary /8 and a more-specific /24 both match).\n\n**Common uses:** routing decisions — given an observed IP, find every VRF, VPC, or security zone it belongs to; determining which policy rules apply to a given address.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "ip", Description: "The IP address to look up."},
			function.ListParameter{
				Name:        "cidrs",
				Description: "The list of CIDRs to search.",
				ElementType: types.StringType,
			},
		},
		Return: function.ListReturn{ElementType: types.StringType},
	}
}

func (f *CIDRsContainingIPFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var ip string
	var cidrs []string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &ip, &cidrs))
	if resp.Error != nil {
		return
	}

	result, err := iputil.CIDRsContainingIP(ip, cidrs)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
