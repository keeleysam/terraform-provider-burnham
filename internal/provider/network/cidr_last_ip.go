package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type CIDRLastIPFunction struct{}

func NewCIDRLastIPFunction() function.Function { return &CIDRLastIPFunction{} }

func (f *CIDRLastIPFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_last_ip"
}

func (f *CIDRLastIPFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Return the last IP address of a CIDR",
		MarkdownDescription: "Returns the last address in the CIDR (all host bits set). For IPv4 this is the broadcast address.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "cidr", Description: "The CIDR to inspect."},
		},
		Return: function.StringReturn{},
	}
}

func (f *CIDRLastIPFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var cidr string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &cidr))
	if resp.Error != nil {
		return
	}

	result, err := iputil.CIDRLastIP(cidr)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
