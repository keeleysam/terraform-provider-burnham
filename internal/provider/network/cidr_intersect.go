package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type CIDRIntersectFunction struct{}

func NewCIDRIntersectFunction() function.Function { return &CIDRIntersectFunction{} }

func (f *CIDRIntersectFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_intersect"
}

func (f *CIDRIntersectFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Return the intersection of two CIDR lists",
		MarkdownDescription: "Returns the set of CIDRs that represent the address space shared between " +
			"list `a` and list `b`. The result is merged into the smallest equivalent set.",
		Parameters: []function.Parameter{
			function.ListParameter{
				Name:        "a",
				Description: "First list of CIDRs.",
				ElementType: types.StringType,
			},
			function.ListParameter{
				Name:        "b",
				Description: "Second list of CIDRs.",
				ElementType: types.StringType,
			},
		},
		Return: function.ListReturn{ElementType: types.StringType},
	}
}

func (f *CIDRIntersectFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var a, b []string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &a, &b))
	if resp.Error != nil {
		return
	}

	result, err := iputil.IntersectCIDRs(a, b)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
