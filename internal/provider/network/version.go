package network

import (
	"context"
	_ "embed"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type IPVersionFunction struct{}

func NewIPVersionFunction() function.Function { return &IPVersionFunction{} }

func (f *IPVersionFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "ip_version"
}

//go:embed descriptions/ip_version.md
var ipVersionDescription string

func (f *IPVersionFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Return the IP version of an address",
		MarkdownDescription: ipVersionDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "ip", Description: "The IP address to inspect."},
		},
		Return: function.Int64Return{},
	}
}

func (f *IPVersionFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var ip string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &ip))
	if resp.Error != nil {
		return
	}

	result, err := iputil.IPVersion(ip)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}

type CIDRVersionFunction struct{}

func NewCIDRVersionFunction() function.Function { return &CIDRVersionFunction{} }

func (f *CIDRVersionFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_version"
}

//go:embed descriptions/cidr_version.md
var cidrVersionDescription string

func (f *CIDRVersionFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Return the IP version of a CIDR",
		MarkdownDescription: cidrVersionDescription,
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
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
