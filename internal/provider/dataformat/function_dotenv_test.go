package dataformat

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runDotenvEncode(t *testing.T, value attr.Value) (string, *function.FuncError) {
	t.Helper()
	f := &DotenvEncodeFunction{}
	args := function.NewArgumentsData([]attr.Value{types.DynamicValue(value)})
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.StringValue(""))}
	f.Run(context.Background(), req, resp)
	if resp.Error != nil {
		return "", resp.Error
	}
	return resp.Result.Value().(types.String).ValueString(), nil
}

func TestValidateDotenvKey(t *testing.T) {
	valid := []string{"FOO", "_PRIVATE", "foo", "FOO_BAR_2"}
	invalid := []string{"", "1FOO", "foo bar", "foo=bar", "foo-bar", "foo.bar", "foo$bar"}
	for _, k := range valid {
		if err := validateDotenvKey(k); err != nil {
			t.Errorf("valid key %q rejected: %v", k, err)
		}
	}
	for _, k := range invalid {
		if err := validateDotenvKey(k); err == nil {
			t.Errorf("invalid key %q accepted", k)
		}
	}
}

func TestDotenvEncode_RejectsInvalidKey(t *testing.T) {
	in := types.ObjectValueMust(
		map[string]attr.Type{"foo bar": types.StringType},
		map[string]attr.Value{"foo bar": types.StringValue("v")},
	)
	_, err := runDotenvEncode(t, in)
	if err == nil {
		t.Error("expected error for invalid key with space")
	}
}

func TestQuoteDotenvValue(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"plain", "plain"},
		{"hello world", `"hello world"`},
		{"a\"b", `"a\"b"`},
		{"a\\b", `"a\\b"`},
		{"a\nb", `"a\nb"`},
		{"with$dollar", `"with$dollar"`},
		{"with#hash", `"with#hash"`},
		{"", ""},
	}
	for _, tc := range cases {
		got := quoteDotenvValue(tc.in)
		if got != tc.want {
			t.Errorf("input %q: want %q, got %q", tc.in, tc.want, got)
		}
	}
}
