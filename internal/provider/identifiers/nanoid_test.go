package identifiers

import (
	"context"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// runNanoidSeedOnly invokes the nanoid function with just a seed (no options) and returns the derived ID.
func runNanoidSeedOnly(t *testing.T, seed string) string {
	t.Helper()
	ctx := context.Background()
	f := NewNanoidFunction()
	args := function.NewArgumentsData([]attr.Value{
		types.StringValue(seed),
		types.TupleValueMust([]attr.Type{}, []attr.Value{}),
	})
	resp := function.RunResponse{Result: function.NewResultData(types.StringNull())}
	f.Run(ctx, function.RunRequest{Arguments: args}, &resp)
	if resp.Error != nil {
		t.Fatalf("nanoid(%q) returned error: %v", seed, resp.Error)
	}
	sv, ok := resp.Result.Value().(types.String)
	if !ok {
		t.Fatalf("nanoid(%q) result was %T, want types.String", seed, resp.Result.Value())
	}
	return sv.ValueString()
}

// docExampleRE pulls the deterministic worked example out of the MarkdownDescription: nanoid("<seed>") → "<expected>".
var docExampleRE = regexp.MustCompile(`nanoid\("([^"]*)"\)\s*→\s*"([^"]+)"`)

// TestNanoid_DocExampleVectorMatches guards the documented example vector against drift: the string shown in the MarkdownDescription must equal what the function actually produces for that seed. If the implementation or the docs change and diverge, this fails.
func TestNanoid_DocExampleVectorMatches(t *testing.T) {
	ctx := context.Background()
	var resp function.DefinitionResponse
	NewNanoidFunction().(*NanoidFunction).Definition(ctx, function.DefinitionRequest{}, &resp)

	m := docExampleRE.FindStringSubmatch(resp.Definition.MarkdownDescription)
	if m == nil {
		t.Fatalf("could not find a `nanoid(\"seed\") → \"result\"` example in the MarkdownDescription")
	}
	seed, documented := m[1], m[2]

	got := runNanoidSeedOnly(t, seed)
	if got != documented {
		t.Fatalf("documented example vector is stale: nanoid(%q) actually returns %q, but the docs claim %q", seed, got, documented)
	}
}
