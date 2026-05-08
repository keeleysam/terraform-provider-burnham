package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type CIDRsAreDisjointFunction struct{}

func NewCIDRsAreDisjointFunction() function.Function { return &CIDRsAreDisjointFunction{} }

func (f *CIDRsAreDisjointFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidrs_are_disjoint"
}

func (f *CIDRsAreDisjointFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Check that no two CIDRs in a list overlap each other",
		MarkdownDescription: "Returns `true` if every CIDR in the list is non-overlapping with every other. Unlike `cidrs_overlap_any`, which compares two separate lists, this checks a single list against itself.\n\n**Common uses:** validating a `list(string)` variable of subnet CIDRs to ensure no two subnets overlap before creating them — catches mistakes like including both a summary prefix and a more-specific one in the same list.",
		Parameters: []function.Parameter{
			function.ListParameter{
				Name:        "cidrs",
				Description: "The list of CIDRs to check.",
				ElementType: types.StringType,
			},
		},
		Return: function.BoolReturn{},
	}
}

func (f *CIDRsAreDisjointFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var cidrs []string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &cidrs))
	if resp.Error != nil {
		return
	}

	result, err := iputil.CIDRsAreDisjoint(cidrs)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
