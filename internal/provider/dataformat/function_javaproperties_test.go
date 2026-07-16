package dataformat

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"unicode/utf16"
	"unicode/utf8"

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

func TestJavaProperties_AstralRoundTrip(t *testing.T) {
	// Astral (non-BMP) runes must be emitted as a UTF-16 surrogate pair of \uXXXX escapes, since each \uXXXX escape is exactly 4 hex digits. The emoji below is U+1F600, which must not be emitted as a single invalid 5-hex-digit escape.
	const emoji = "\U0001F600"
	cases := map[string]struct{ in, want string }{
		"value": {"a" + emoji + "b", "a\\uD83D\\uDE00b"},
		"key":   {"k" + emoji + "y", "k\\uD83D\\uDE00y"},
	}
	escape := map[string]func(string) string{
		"value": escapeJavaPropertiesValue,
		"key":   escapeJavaPropertiesKey,
	}
	for name, tc := range cases {
		got := escape[name](tc.in)
		if got != tc.want {
			t.Errorf("%s escape: want %q, got %q", name, tc.want, got)
		}
		if strings.Contains(got, "1F600") {
			t.Errorf("%s escape emitted invalid 5-hex \\u1F600 sequence: %q", name, got)
		}
		// A conformant reader unescapes each \uXXXX into a UTF-16 code unit, then decodes the pair back to the original rune.
		if rt := javaUnescapeForTest(got); rt != tc.in {
			t.Errorf("%s did not round-trip through a conformant reader: want %q, got %q", name, tc.in, rt)
		}
	}
}

// javaUnescapeForTest models a conformant .properties reader: it turns each \uXXXX escape into a UTF-16 code unit and passes literal bytes through, then decodes the resulting UTF-16 sequence (recombining surrogate pairs) into a Go string.
func javaUnescapeForTest(s string) string {
	var units []uint16
	for i := 0; i < len(s); {
		if i+5 < len(s) && s[i] == '\\' && s[i+1] == 'u' {
			v, err := strconv.ParseUint(s[i+2:i+6], 16, 16)
			if err != nil {
				panic(err)
			}
			units = append(units, uint16(v))
			i += 6
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		units = append(units, utf16.Encode([]rune{r})...)
		i += size
	}
	return string(utf16.Decode(units))
}

func TestEscapeJavaPropertiesValue_LeadingSpace(t *testing.T) {
	// Leading whitespace must be escaped because .properties parsers strip leading whitespace from values.
	got := escapeJavaPropertiesValue("  v")
	want := `\ \ v`
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}
