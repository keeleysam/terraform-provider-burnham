package text

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// defaultParseKVOpts mirrors the documented defaults so the pure-function tests
// can override just the field under test.
func defaultParseKVOpts() parseKVOpts {
	return parseKVOpts{pairSep: ",", kvSep: "=", trim: true, unquote: true}
}

func TestParseKV_Core(t *testing.T) {
	cases := []struct {
		name string
		in   string
		opts parseKVOpts
		want map[string]string
	}{
		{
			name: "basic",
			in:   "a=1,b=2",
			opts: defaultParseKVOpts(),
			want: map[string]string{"a": "1", "b": "2"},
		},
		{
			name: "equals in value splits on first kv_sep only",
			in:   "url=https://x?a=b",
			opts: defaultParseKVOpts(),
			want: map[string]string{"url": "https://x?a=b"},
		},
		{
			name: "a=b=c keeps trailing kv_sep in value",
			in:   "a=b=c",
			opts: defaultParseKVOpts(),
			want: map[string]string{"a": "b=c"},
		},
		{
			name: "whitespace trimmed by default",
			in:   "a = 1, b = 2",
			opts: defaultParseKVOpts(),
			want: map[string]string{"a": "1", "b": "2"},
		},
		{
			name: "whitespace kept when trim disabled",
			in:   "a = 1, b = 2",
			opts: parseKVOpts{pairSep: ",", kvSep: "=", trim: false, unquote: true},
			want: map[string]string{"a ": " 1", " b ": " 2"},
		},
		{
			name: "double-quoted value protects pair separator",
			in:   `a="x,y",b=2`,
			opts: defaultParseKVOpts(),
			want: map[string]string{"a": "x,y", "b": "2"},
		},
		{
			name: "single-quoted value protects pair separator",
			in:   `a='x,y',b=2`,
			opts: defaultParseKVOpts(),
			want: map[string]string{"a": "x,y", "b": "2"},
		},
		{
			name: "quotes literal when unquote disabled",
			in:   `a="x",b='y'`,
			opts: parseKVOpts{pairSep: ",", kvSep: "=", trim: true, unquote: false},
			want: map[string]string{"a": `"x"`, "b": `'y'`},
		},
		{
			name: "custom separators",
			in:   "a:1;b:2",
			opts: parseKVOpts{pairSep: ";", kvSep: ":", trim: true, unquote: true},
			want: map[string]string{"a": "1", "b": "2"},
		},
		{
			name: "empty value",
			in:   "a=",
			opts: defaultParseKVOpts(),
			want: map[string]string{"a": ""},
		},
		{
			name: "empty quoted value unwraps to empty string",
			in:   `a=""`,
			opts: defaultParseKVOpts(),
			want: map[string]string{"a": ""},
		},
		{
			name: "skip empty segments",
			in:   "a=1,,b=2,",
			opts: defaultParseKVOpts(),
			want: map[string]string{"a": "1", "b": "2"},
		},
		{
			name: "whitespace-only segment skipped when trimming",
			in:   "a=1, ,b=2",
			opts: defaultParseKVOpts(),
			want: map[string]string{"a": "1", "b": "2"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parseKV(c.in, c.opts)
			if err != nil {
				t.Fatalf("parseKV(%q) unexpected error: %v", c.in, err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("parseKV(%q) = %#v, want %#v", c.in, got, c.want)
			}
		})
	}
}

func TestParseKV_CoreErrors(t *testing.T) {
	cases := []struct {
		name string
		in   string
		opts parseKVOpts
	}{
		{
			name: "pair without kv_sep",
			in:   "a=1,bogus",
			opts: defaultParseKVOpts(),
		},
		{
			name: "duplicate key",
			in:   "a=1,a=2",
			opts: defaultParseKVOpts(),
		},
		{
			name: "unquote disabled exposes the naive-split breakage",
			in:   `a="x,y",b=2`,
			opts: parseKVOpts{pairSep: ",", kvSep: "=", trim: true, unquote: false},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := parseKV(c.in, c.opts); err == nil {
				t.Fatalf("parseKV(%q) expected an error, got nil", c.in)
			}
		})
	}
}

// runParseKV drives the function end to end through the framework and returns the
// result value plus any function error.
func runParseKV(t *testing.T, args ...attr.Value) (attr.Value, *function.FuncError) {
	t.Helper()
	f := NewParseKVFunction()
	resp := function.RunResponse{Result: function.NewResultData(types.MapUnknown(types.StringType))}
	f.Run(context.Background(), function.RunRequest{Arguments: function.NewArgumentsData(args)}, &resp)
	return resp.Result.Value(), resp.Error
}

func mapResultToGo(t *testing.T, v attr.Value) map[string]string {
	t.Helper()
	mv, ok := v.(basetypes.MapValue)
	if !ok {
		t.Fatalf("result was %T, want basetypes.MapValue", v)
	}
	out := make(map[string]string, len(mv.Elements()))
	for k, ev := range mv.Elements() {
		sv, ok := ev.(basetypes.StringValue)
		if !ok {
			t.Fatalf("element %q was %T, want basetypes.StringValue", k, ev)
		}
		out[k] = sv.ValueString()
	}
	return out
}

func noOpts() attr.Value {
	return types.TupleValueMust([]attr.Type{}, []attr.Value{})
}

func optsObject(t *testing.T, attrs map[string]attr.Value) attr.Value {
	t.Helper()
	types_ := make(map[string]attr.Type, len(attrs))
	for k, v := range attrs {
		types_[k] = v.Type(context.Background())
	}
	obj := types.ObjectValueMust(types_, attrs)
	return types.TupleValueMust([]attr.Type{types.DynamicType}, []attr.Value{types.DynamicValue(obj)})
}

func TestParseKV_RunBasic(t *testing.T) {
	v, ferr := runParseKV(t, types.StringValue("a=1,b=2"), noOpts())
	if ferr != nil {
		t.Fatalf("unexpected error: %v", ferr)
	}
	got := mapResultToGo(t, v)
	want := map[string]string{"a": "1", "b": "2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestParseKV_RunCustomSeparators(t *testing.T) {
	opts := optsObject(t, map[string]attr.Value{
		"pair_sep": types.StringValue(";"),
		"kv_sep":   types.StringValue(":"),
	})
	v, ferr := runParseKV(t, types.StringValue("a:1;b:2"), opts)
	if ferr != nil {
		t.Fatalf("unexpected error: %v", ferr)
	}
	got := mapResultToGo(t, v)
	want := map[string]string{"a": "1", "b": "2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestParseKV_RunNullOptions(t *testing.T) {
	// A null options object falls through to defaults, matching SingleOptionsObject's contract via the wider function family.
	nullOpts := types.TupleValueMust(
		[]attr.Type{types.DynamicType},
		[]attr.Value{types.DynamicValue(types.ObjectNull(map[string]attr.Type{"trim": types.BoolType}))},
	)
	_, ferr := runParseKV(t, types.StringValue("a=1"), nullOpts)
	if ferr == nil {
		t.Fatalf("expected an error for a null options object")
	}
}

func TestParseKV_RunUnknownOptionYieldsUnknown(t *testing.T) {
	opts := optsObject(t, map[string]attr.Value{
		"pair_sep": types.StringUnknown(),
	})
	v, ferr := runParseKV(t, types.StringValue("a=1,b=2"), opts)
	if ferr != nil {
		t.Fatalf("unexpected error: %v", ferr)
	}
	if !v.IsUnknown() {
		t.Fatalf("expected an unknown result, got %#v", v)
	}
}

func TestParseKV_RunErrors(t *testing.T) {
	if _, ferr := runParseKV(t, types.StringValue("a=1,bogus"), noOpts()); ferr == nil {
		t.Fatalf("expected an error for a pair with no kv_sep")
	}
	if _, ferr := runParseKV(t, types.StringValue("a=1,a=2"), noOpts()); ferr == nil {
		t.Fatalf("expected an error for a duplicate key")
	}
}
