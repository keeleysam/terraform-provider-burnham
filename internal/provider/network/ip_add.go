package network

import (
	"context"
	_ "embed"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type IPAddFunction struct{}

func NewIPAddFunction() function.Function { return &IPAddFunction{} }

func (f *IPAddFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "ip_add"
}

//go:embed descriptions/ip_add.md
var ipAddDescription string

func (f *IPAddFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Offset an IP address by an integer",
		MarkdownDescription: ipAddDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "ip", Description: "The base IP address."},
			function.Int64Parameter{Name: "n", Description: "The integer offset to apply (may be negative)."},
		},
		Return: function.StringReturn{},
	}
}

func (f *IPAddFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var ip string
	var n int64
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &ip, &n))
	if resp.Error != nil {
		return
	}

	result, err := iputil.IPAdd(ip, n)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
