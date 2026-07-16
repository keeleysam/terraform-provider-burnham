package network

import (
	"context"
	_ "embed"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type CIDRHostCountFunction struct{}

func NewCIDRHostCountFunction() function.Function { return &CIDRHostCountFunction{} }

func (f *CIDRHostCountFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_host_count"
}

//go:embed descriptions/cidr_host_count.md
var cidrHostCountDescription string

func (f *CIDRHostCountFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Return the total number of addresses in a CIDR",
		MarkdownDescription: cidrHostCountDescription,
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
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}

type CIDRUsableHostCountFunction struct{}

func NewCIDRUsableHostCountFunction() function.Function { return &CIDRUsableHostCountFunction{} }

func (f *CIDRUsableHostCountFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_usable_host_count"
}

//go:embed descriptions/cidr_usable_host_count.md
var cidrUsableHostCountDescription string

func (f *CIDRUsableHostCountFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Return the number of usable host addresses in a CIDR",
		MarkdownDescription: cidrUsableHostCountDescription,
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
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
