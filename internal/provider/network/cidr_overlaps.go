package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type CIDROverlapsFunction struct{}

func NewCIDROverlapsFunction() function.Function { return &CIDROverlapsFunction{} }

func (f *CIDROverlapsFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_overlaps"
}

func (f *CIDROverlapsFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Check whether two CIDRs share any addresses",
		MarkdownDescription: "Returns `true` if CIDR `a` and CIDR `b` overlap (share at least one address).",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "a", Description: "First CIDR."},
			function.StringParameter{Name: "b", Description: "Second CIDR."},
		},
		Return: function.BoolReturn{},
	}
}

func (f *CIDROverlapsFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var a, b string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &a, &b))
	if resp.Error != nil {
		return
	}

	result, err := iputil.CIDROverlaps(a, b)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
