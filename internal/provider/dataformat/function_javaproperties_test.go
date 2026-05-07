package dataformat

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runJavaPropertiesDecode(t *testing.T, input string) (types.Dynamic, *function.FuncError) {
	t.Helper()
	f := &JavaPropertiesDecodeFunction{}
	args := function.NewArgumentsData([]attr.Value{types.StringValue(input)})
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.DynamicNull())}
	f.Run(context.Background(), req, resp)
	if resp.Error != nil {
		return types.DynamicNull(), resp.Error
	}
	return resp.Result.Value().(types.Dynamic), nil
}

func TestJavaPropertiesDecode_LineContinuation(t *testing.T) {
	got, err := runJavaPropertiesDecode(t, "key=line1 \\\n  line2\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v := got.UnderlyingValue().(types.Object).Attributes()["key"].(types.String).ValueString()
	if v != "line1 line2" {
		t.Errorf("want continuation to merge, got %q", v)
	}
}

func TestJavaPropertiesDecode_ColonSeparator(t *testing.T) {
	got, err := runJavaPropertiesDecode(t, "k:v\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v := got.UnderlyingValue().(types.Object).Attributes()["k"].(types.String).ValueString()
	if v != "v" {
		t.Errorf("want %q, got %q", "v", v)
	}
}

func TestJavaPropertiesDecode_BangComments(t *testing.T) {
	got, err := runJavaPropertiesDecode(t, "! comment line\nk=v\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	attrs := got.UnderlyingValue().(types.Object).Attributes()
	if _, has := attrs["! comment line"]; has {
		t.Error("comment leaked into output")
	}
	if attrs["k"].(types.String).ValueString() != "v" {
		t.Error("missing real key")
	}
}

func TestJavaPropertiesDecode_NoExpansion(t *testing.T) {
	got, err := runJavaPropertiesDecode(t, "a=hello\nb=${a} world\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v := got.UnderlyingValue().(types.Object).Attributes()["b"].(types.String).ValueString()
	if v != "${a} world" {
		t.Errorf("expected ${a} expansion to be disabled, got %q", v)
	}
}

func TestEscapeJavaPropertiesKey_Specials(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"plain", "plain"},
		{"a=b", `a\=b`},
		{"a:b", `a\:b`},
		{"a b", `a\ b`},
		{"a#b", `a\#b`},
		{"a!b", `a\!b`},
		{"a\\b", `a\\b`},
	}
	for _, tc := range cases {
		got := escapeJavaPropertiesKey(tc.in)
		if got != tc.want {
			t.Errorf("input %q: want %q, got %q", tc.in, tc.want, got)
		}
	}
}

func TestEscapeJavaPropertiesValue_NonASCII(t *testing.T) {
	// Non-ASCII characters are emitted as \uXXXX escapes for portability with legacy ISO-8859-1 readers (this is a documented choice; see comment on renderJavaProperties).
	got := escapeJavaPropertiesValue("hellö")
	want := `hell\u00F6`
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestEscapeJavaPropertiesValue_LeadingSpace(t *testing.T) {
	// Leading whitespace must be escaped because .properties parsers strip leading whitespace from values.
	got := escapeJavaPropertiesValue("  v")
	want := `\ \ v`
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}
