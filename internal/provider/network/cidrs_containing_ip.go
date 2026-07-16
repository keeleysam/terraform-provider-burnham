package network

import (
	"context"
	_ "embed"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type CIDRsContainingIPFunction struct{}

func NewCIDRsContainingIPFunction() function.Function { return &CIDRsContainingIPFunction{} }

func (f *CIDRsContainingIPFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidrs_containing_ip"
}

//go:embed descriptions/cidrs_containing_ip.md
var cidrsContainingIpDescription string

func (f *CIDRsContainingIPFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Return every CIDR from a list that contains the given IP address",
		MarkdownDescription: cidrsContainingIpDescription,
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
	var cidrsArg types.List
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &ip, &cidrsArg))
	if resp.Error != nil {
		return
	}
	cidrs, argErr := cidrListArg(cidrsArg, 1, "cidrs")
	if argErr != nil {
		resp.Error = argErr
		return
	}

	result, err := iputil.CIDRsContainingIP(ip, cidrs)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
