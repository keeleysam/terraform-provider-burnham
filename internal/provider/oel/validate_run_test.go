package oel

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestValidateRun_OversizeReturnsFalse locks the "never fails the plan" contract
// that oelvalidate's description promises: on input past the size guard it must
// return false, not a plan-failing argument error, matching promqlvalidate.
func TestValidateRun_OversizeReturnsFalse(t *testing.T) {
	big := strings.Repeat("a", oelMaxInputBytes+1)
	f := &OELValidateFunction{}
	args := function.NewArgumentsData([]attr.Value{types.StringValue(big)})
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.BoolValue(true))}

	f.Run(context.Background(), req, resp)

	if resp.Error != nil {
		t.Fatalf("oversize input must not fail the plan, got error: %v", resp.Error)
	}
	got, ok := resp.Result.Value().(types.Bool)
	if !ok {
		t.Fatalf("expected Bool result, got %T", resp.Result.Value())
	}
	if got.ValueBool() {
		t.Fatalf("oversize input = valid true, want false")
	}
}
