package network

import (
	"context"
	_ "embed"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type CIDRFindFreeFunction struct{}

func NewCIDRFindFreeFunction() function.Function { return &CIDRFindFreeFunction{} }

func (f *CIDRFindFreeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_find_free"
}

//go:embed descriptions/cidr_find_free.md
var cidrFindFreeDescription string

func (f *CIDRFindFreeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Find the first available subnet of a given prefix length within a pool",
		MarkdownDescription: cidrFindFreeDescription,
		Parameters: []function.Parameter{
			function.ListParameter{
				Name:        "pool",
				Description: "The address space to allocate from.",
				ElementType: types.StringType,
			},
			function.ListParameter{
				Name:        "used",
				Description: "CIDRs already in use within the pool.",
				ElementType: types.StringType,
			},
			function.Int64Parameter{
				Name:        "prefix_len",
				Description: "The desired prefix length of the returned subnet (e.g. 24 for a /24).",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *CIDRFindFreeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var poolArg, usedArg types.List
	var prefixLen int64
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &poolArg, &usedArg, &prefixLen))
	if resp.Error != nil {
		return
	}
	pool, argErr := cidrListArg(poolArg, 0, "pool")
	if argErr != nil {
		resp.Error = argErr
		return
	}
	used, argErr := cidrListArg(usedArg, 1, "used")
	if argErr != nil {
		resp.Error = argErr
		return
	}

	result, err := iputil.FindFreeCIDR(pool, used, prefixLen)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	if result == nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.StringNull()))
	} else {
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.StringValue(*result)))
	}
}
