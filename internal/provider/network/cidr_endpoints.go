package network

import (
	"context"
	_ "embed"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type CIDRFirstIPFunction struct{}

func NewCIDRFirstIPFunction() function.Function { return &CIDRFirstIPFunction{} }

func (f *CIDRFirstIPFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_first_ip"
}

//go:embed descriptions/cidr_first_ip.md
var cidrFirstIpDescription string

func (f *CIDRFirstIPFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Return the first (network) IP address of a CIDR",
		MarkdownDescription: cidrFirstIpDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "cidr", Description: "The CIDR to inspect."},
		},
		Return: function.StringReturn{},
	}
}

func (f *CIDRFirstIPFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var cidr string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &cidr))
	if resp.Error != nil {
		return
	}

	result, err := iputil.CIDRFirstIP(cidr)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}

type CIDRLastIPFunction struct{}

func NewCIDRLastIPFunction() function.Function { return &CIDRLastIPFunction{} }

func (f *CIDRLastIPFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_last_ip"
}

//go:embed descriptions/cidr_last_ip.md
var cidrLastIpDescription string

func (f *CIDRLastIPFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Return the last IP address of a CIDR",
		MarkdownDescription: cidrLastIpDescription,
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
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
