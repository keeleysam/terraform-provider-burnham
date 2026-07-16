package network

import (
	"context"
	_ "embed"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type IPSubtractFunction struct{}

func NewIPSubtractFunction() function.Function { return &IPSubtractFunction{} }

func (f *IPSubtractFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "ip_subtract"
}

//go:embed descriptions/ip_subtract.md
var ipSubtractDescription string

func (f *IPSubtractFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Return the signed integer distance between two IP addresses",
		MarkdownDescription: ipSubtractDescription,
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
