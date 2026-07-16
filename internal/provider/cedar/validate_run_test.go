package cedar

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestValidateRun_OversizeReturnsFalse locks the "never fails the plan" contract
// that cedarvalidate's description promises: on input past the size guard it must
// return false, not a plan-failing argument error, matching promqlvalidate.
func TestValidateRun_OversizeReturnsFalse(t *testing.T) {
	// Use content that is valid Cedar so this isolates the size guard, not the parser: without the guard, cedar-go would parse this repeated policy set and return true. (A run of "a" would return false regardless of the guard, so it would not catch the guard's removal.)
	unit := "permit ( principal, action, resource );\n"
	big := strings.Repeat(unit, cedarMaxInputBytes/len(unit)+2)
	f := &CedarValidateFunction{}
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
