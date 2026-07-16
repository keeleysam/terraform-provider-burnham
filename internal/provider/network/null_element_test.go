package network

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// cidrListWithNull builds a list(string) whose second element is null, the
// shape that a Terraform expression like ["10.0.0.0/8", null] produces.
func cidrListWithNull() types.List {
	return types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("10.0.0.0/8"),
		types.StringNull(),
	})
}

// assertArgumentError fails unless err is a non-nil argument-attributed error
// pointing at wantArg, rather than the framework's provider-blaming internal
// conversion error.
func assertArgumentError(t *testing.T, err *function.FuncError, wantArg int64) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected an error for a null list element, got nil")
	}
	if strings.Contains(err.Text, "report to the provider developers") {
		t.Fatalf("got provider-blaming error instead of an argument error: %q", err.Text)
	}
	if err.FunctionArgument == nil {
		t.Fatalf("expected error attributed to an argument, got FunctionArgument=nil (text: %q)", err.Text)
	}
	if int64(*err.FunctionArgument) != wantArg {
		t.Fatalf("expected error attributed to argument %d, got %d", wantArg, *err.FunctionArgument)
	}
}

func TestCIDRsAreDisjoint_NullElement(t *testing.T) {
	f := &CIDRsAreDisjointFunction{}
	req := function.RunRequest{Arguments: function.NewArgumentsData([]attr.Value{cidrListWithNull()})}
	resp := &function.RunResponse{Result: function.NewResultData(types.BoolValue(false))}

	f.Run(context.Background(), req, resp)

	assertArgumentError(t, resp.Error, 0)
}

func TestCIDRsOverlapAny_NullElementInSecondList(t *testing.T) {
	f := &CIDRsOverlapAnyFunction{}
	good := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("10.0.0.0/8")})
	req := function.RunRequest{Arguments: function.NewArgumentsData([]attr.Value{good, cidrListWithNull()})}
	resp := &function.RunResponse{Result: function.NewResultData(types.BoolValue(false))}

	f.Run(context.Background(), req, resp)

	assertArgumentError(t, resp.Error, 1)
}
