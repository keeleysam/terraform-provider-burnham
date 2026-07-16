package cedar

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestEncodeRun_ValidPolicy covers the Run success plumbing end to end
// (terraformToNode -> Encode -> resp.Result.Set), which the error-path test
// alone does not exercise. The EST attr value is produced from a known policy
// via Decode + nodeToAttr, mirroring what promqldecode/cedardecode hand back.
func TestEncodeRun_ValidPolicy(t *testing.T) {
	tree, err := Decode("permit ( principal, action, resource );")
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	av, err := nodeToAttr(tree)
	if err != nil {
		t.Fatalf("nodeToAttr: %v", err)
	}
	f := &CedarEncodeFunction{}
	args := function.NewArgumentsData([]attr.Value{types.DynamicValue(av)})
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.StringValue(""))}

	f.Run(context.Background(), req, resp)

	if resp.Error != nil {
		t.Fatalf("valid EST should encode, got error: %v", resp.Error)
	}
	got, ok := resp.Result.Value().(types.String)
	if !ok {
		t.Fatalf("expected String result, got %T", resp.Result.Value())
	}
	if !strings.HasPrefix(got.ValueString(), "permit") {
		t.Fatalf("encoded policy should start with permit, got %q", got.ValueString())
	}
}

// TestEncodeRun_InvalidESTIsArgumentError locks that a structurally-valid tree
// that is not a valid Cedar policy (reachable bad user input) is attributed to
// argument 0, matching how celencode/oelencode/promqlencode report bad input.
// The errInvalidOutput sentinel is reserved for a genuinely-internal failure.
func TestEncodeRun_InvalidESTIsArgumentError(t *testing.T) {
	tree := types.ObjectValueMust(
		map[string]attr.Type{"effect": types.StringType},
		map[string]attr.Value{"effect": types.StringValue("maybe")},
	)
	f := &CedarEncodeFunction{}
	args := function.NewArgumentsData([]attr.Value{types.DynamicValue(tree)})
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.StringValue(""))}

	f.Run(context.Background(), req, resp)

	if resp.Error == nil {
		t.Fatal("invalid EST should error")
	}
	if resp.Error.FunctionArgument == nil || *resp.Error.FunctionArgument != 0 {
		t.Fatalf("invalid EST should be attributed to argument 0, got FunctionArgument=%v (%q)", resp.Error.FunctionArgument, resp.Error.Text)
	}
}
