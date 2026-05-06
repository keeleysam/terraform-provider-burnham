package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type IPSubtractFunction struct{}

func NewIPSubtractFunction() function.Function { return &IPSubtractFunction{} }

func (f *IPSubtractFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "ip_subtract"
}

func (f *IPSubtractFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Return the signed integer distance between two IP addresses",
		MarkdownDescription: "Returns `a - b` as an integer: how many address positions separate the two IPs. " +
			"A positive result means `a` is higher; negative means `b` is higher; zero means they are equal. " +
			"Both addresses must be the same family. For IPv4 the result always fits; for IPv6 an error is " +
			"returned if the difference exceeds int64 range.\n\n" +
			"**Common uses:** asserting that two IPs are within N addresses of each other; computing the " +
			"length of an arbitrary IP range; confirming an IP falls exactly at the expected offset from " +
			"a base address; generating loop indices over a sparse range.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "a", Description: "The IP to subtract from."},
			function.StringParameter{Name: "b", Description: "The IP to subtract."},
		},
		Return: function.Int64Return{},
	}
}

func (f *IPSubtractFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var a, b string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &a, &b))
	if resp.Error != nil {
		return
	}

	result, err := iputil.IPSubtract(a, b)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
