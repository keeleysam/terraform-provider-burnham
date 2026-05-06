package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type IPIsPrivateFunction struct{}

func NewIPIsPrivateFunction() function.Function { return &IPIsPrivateFunction{} }

func (f *IPIsPrivateFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "ip_is_private"
}

func (f *IPIsPrivateFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Check whether an IP address is in a private range",
		MarkdownDescription: "Returns `true` if the IP address is within a private, loopback, " +
			"link-local, or CGNAT range (RFC1918, RFC6598, RFC4193, loopback, link-local).",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "ip", Description: "The IP address to check."},
		},
		Return: function.BoolReturn{},
	}
}

func (f *IPIsPrivateFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var ip string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &ip))
	if resp.Error != nil {
		return
	}

	result, err := iputil.IPIsPrivate(ip)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
