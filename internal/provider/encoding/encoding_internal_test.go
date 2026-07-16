package encoding

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// runEncoder invokes an encode/decode function with an input string and a single
// options object, returning the result value so the caller can assert on its
// unknown-ness.
func runEncoder(t *testing.T, f function.Function, input string, opts attr.Value) attr.Value {
	t.Helper()
	args := function.NewArgumentsData([]attr.Value{
		types.StringValue(input),
		types.TupleValueMust([]attr.Type{types.DynamicType}, []attr.Value{types.DynamicValue(opts)}),
	})
	resp := &function.RunResponse{Result: function.NewResultData(types.StringValue(""))}
	f.Run(context.Background(), function.RunRequest{Arguments: args}, resp)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	return resp.Result.Value()
}

func TestBase64Encode_UnknownOptionReturnsUnknown(t *testing.T) {
	opts := types.ObjectValueMust(
		map[string]attr.Type{"url_safe": types.BoolType},
		map[string]attr.Value{"url_safe": types.BoolUnknown()},
	)
	if got := runEncoder(t, NewBase64EncodeFunction(), "Hello", opts); !got.IsUnknown() {
		t.Fatalf("expected unknown result for unknown url_safe, got %#v", got)
	}
}

func TestBase32Encode_UnknownOptionReturnsUnknown(t *testing.T) {
	opts := types.ObjectValueMust(
		map[string]attr.Type{"padding": types.BoolType},
		map[string]attr.Value{"padding": types.BoolUnknown()},
	)
	if got := runEncoder(t, NewBase32EncodeFunction(), "foobar", opts); !got.IsUnknown() {
		t.Fatalf("expected unknown result for unknown padding, got %#v", got)
	}
}

func TestBase32Decode_UnknownOptionReturnsUnknown(t *testing.T) {
	opts := types.ObjectValueMust(
		map[string]attr.Type{"hex_alphabet": types.BoolType},
		map[string]attr.Value{"hex_alphabet": types.BoolUnknown()},
	)
	if got := runEncoder(t, NewBase32DecodeFunction(), "MZXW6YTBOI", opts); !got.IsUnknown() {
		t.Fatalf("expected unknown result for unknown hex_alphabet, got %#v", got)
	}
}

func TestHexEncode_Known(t *testing.T) {
	if got := hexEncode([]byte("Hi")); got != "4869" {
		t.Errorf("hexEncode(\"Hi\") = %q, want \"4869\"", got)
	}
}

func TestHexDecode_Lenient(t *testing.T) {
	cases := []string{"4869", "48 69", "48\n69", "4869", "4869"}
	for _, in := range cases {
		got, err := hexDecodeLenient(in)
		if err != nil {
			t.Fatalf("hexDecodeLenient(%q): %v", in, err)
		}
		if string(got) != "Hi" {
			t.Errorf("hexDecodeLenient(%q) = %q, want \"Hi\"", in, got)
		}
	}
}

func TestHexDecode_CaseInsensitive(t *testing.T) {
	lo, err := hexDecodeLenient("deadbeef")
	if err != nil {
		t.Fatal(err)
	}
	up, err := hexDecodeLenient("DEADBEEF")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(lo, up) {
		t.Errorf("hex decode is case-sensitive: %x vs %x", lo, up)
	}
}

func TestHexDecode_Errors(t *testing.T) {
	if _, err := hexDecodeLenient("abc"); err == nil {
		t.Error("expected odd-length error")
	}
	if _, err := hexDecodeLenient("zz"); err == nil {
		t.Error("expected invalid-char error")
	}
}

func TestBase64Encode_DefaultMatchesStd(t *testing.T) {
	if got := base64Encode([]byte("Hello"), false, true); got != "SGVsbG8=" {
		t.Errorf("base64Encode default = %q, want \"SGVsbG8=\"", got)
	}
}

func TestBase64Encode_URLSafeAlphabet(t *testing.T) {
	// Bytes chosen so standard base64 yields both '+' and '/'.
	in := []byte{0xfb, 0xff, 0xbf, 0xfe}
	std := base64Encode(in, false, true)
	url := base64Encode(in, true, true)
	if !strings.ContainsAny(std, "+/") {
		t.Fatalf("test input did not exercise +/ in standard alphabet: %q", std)
	}
	if strings.ContainsAny(url, "+/") {
		t.Errorf("url-safe output contains +/: %q", url)
	}
}

func TestBase64Encode_NoPadding(t *testing.T) {
	if got := base64Encode([]byte("Hello"), false, false); strings.Contains(got, "=") {
		t.Errorf("padding=false still produced '=': %q", got)
	}
}

func TestBase64Decode_LenientAcceptsAllVariants(t *testing.T) {
	in := []byte{0xfb, 0xff, 0xbf, 0xfe, 0x00, 0x10}
	for _, urlSafe := range []bool{false, true} {
		for _, padding := range []bool{false, true} {
			enc := base64Encode(in, urlSafe, padding)
			got, err := base64DecodeLenient(enc)
			if err != nil {
				t.Fatalf("decode(%q) [url=%v pad=%v]: %v", enc, urlSafe, padding, err)
			}
			if !bytes.Equal(got, in) {
				t.Errorf("round-trip mismatch [url=%v pad=%v]: %x != %x", urlSafe, padding, got, in)
			}
		}
	}
}

func TestBase64Decode_IgnoresWhitespace(t *testing.T) {
	got, err := base64DecodeLenient("SGVs\nbG8=")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "Hello" {
		t.Errorf("got %q, want \"Hello\"", got)
	}
}

func TestBase64Decode_Invalid(t *testing.T) {
	if _, err := base64DecodeLenient("not valid base64!@#$"); err == nil {
		t.Error("expected error for invalid base64")
	}
}

// ─── base32 (RFC 4648 §10 test vectors) ─────────────────────────

func TestBase32Encode_RFCVectors(t *testing.T) {
	if got := base32Encode([]byte("foobar"), false, true); got != "MZXW6YTBOI======" {
		t.Errorf("std padded = %q, want MZXW6YTBOI======", got)
	}
	if got := base32Encode([]byte("foobar"), false, false); got != "MZXW6YTBOI" {
		t.Errorf("std unpadded = %q, want MZXW6YTBOI", got)
	}
	if got := base32Encode([]byte("foobar"), true, true); got != "CPNMUOJ1E8======" {
		t.Errorf("hex padded = %q, want CPNMUOJ1E8======", got)
	}
}

func TestBase32Decode_Lenient(t *testing.T) {
	// padded, unpadded, lowercase (case-insensitive), and with whitespace all → "foobar"
	for _, in := range []string{"MZXW6YTBOI======", "MZXW6YTBOI", "mzxw6ytboi", "MZXW 6YTB OI"} {
		got, err := base32DecodeLenient(in, false)
		if err != nil {
			t.Fatalf("base32DecodeLenient(%q): %v", in, err)
		}
		if string(got) != "foobar" {
			t.Errorf("base32DecodeLenient(%q) = %q, want foobar", in, got)
		}
	}
}

func TestBase32Decode_HexAlphabet(t *testing.T) {
	got, err := base32DecodeLenient("CPNMUOJ1E8======", true)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "foobar" {
		t.Errorf("got %q, want foobar", got)
	}
}

func TestBase32_RoundTripAllVariants(t *testing.T) {
	in := []byte{0x00, 0xff, 0x10, 0x80, 0x7f, 0x01, 0x02}
	for _, hexAlpha := range []bool{false, true} {
		for _, pad := range []bool{false, true} {
			enc := base32Encode(in, hexAlpha, pad)
			got, err := base32DecodeLenient(enc, hexAlpha)
			if err != nil {
				t.Fatalf("decode(%q) [hex=%v pad=%v]: %v", enc, hexAlpha, pad, err)
			}
			if !bytes.Equal(got, in) {
				t.Errorf("round-trip mismatch [hex=%v pad=%v]: %x != %x", hexAlpha, pad, got, in)
			}
		}
	}
}

func TestBase32Decode_Invalid(t *testing.T) {
	if _, err := base32DecodeLenient("0189", false); err == nil {
		t.Error("expected error: 0/1/8/9 are not in the standard base32 alphabet")
	}
}

func TestBase32Decode_RejectsUnicodeHomoglyphs(t *testing.T) {
	// Uppercasing must be ASCII-only. Unicode case folding would fold U+0131
	// (ı, dotless i) into 'I' and U+017F (ſ, long s) into 'S', both valid
	// base32 letters, letting non-alphabet input decode instead of erroring.
	for _, in := range []string{"MZXW6YTBOı", "ıſıſıſıſ"} {
		if _, err := base32DecodeLenient(in, false); err == nil {
			t.Errorf("base32DecodeLenient(%q): expected error for non-alphabet Unicode input", in)
		}
	}
}

// ─── url encode / decode ────────────────────────────────────────

func TestURLEncode_Modes(t *testing.T) {
	cases := []struct {
		mode, in, want string
	}{
		{"query", "a b+c/d", "a+b%2Bc%2Fd"},       // form: space→+, +→%2B, /→%2F
		{"path", "a b+c/d", "a%20b+c%2Fd"},        // segment: space→%20, + literal, /→%2F
		{"component", "a b+c/d", "a%20b%2Bc%2Fd"}, // strict: everything non-unreserved escaped
	}
	for _, c := range cases {
		if got := urlEncode(c.in, c.mode); got != c.want {
			t.Errorf("urlEncode(%q, %q) = %q, want %q", c.in, c.mode, got, c.want)
		}
	}
}

func TestURLEncode_QueryMatchesCoreDefault(t *testing.T) {
	// query mode is the drop-in for core's urlencode (application/x-www-form-urlencoded).
	if got := urlEncode("a b", "query"); got != "a+b" {
		t.Errorf("got %q, want a+b", got)
	}
}

func TestURLDecode_PlusAmbiguity(t *testing.T) {
	// The reason decode needs a mode: + means space in query, literal in path.
	if got, _ := urlDecode("1+1", "query"); got != "1 1" {
		t.Errorf("query: got %q, want \"1 1\"", got)
	}
	if got, _ := urlDecode("1+1", "path"); got != "1+1" {
		t.Errorf("path: got %q, want \"1+1\"", got)
	}
}

func TestURLDecode_PercentAndRoundTrip(t *testing.T) {
	for _, mode := range []string{"query", "path", "component"} {
		enc := urlEncode("hello world/ä?&=", mode)
		got, err := urlDecode(enc, mode)
		if err != nil {
			t.Fatalf("urlDecode(%q, %q): %v", enc, mode, err)
		}
		if got != "hello world/ä?&=" {
			t.Errorf("round-trip [%s] = %q", mode, got)
		}
	}
}

func TestURLDecode_Invalid(t *testing.T) {
	if _, err := urlDecode("%ZZ", "query"); err == nil {
		t.Error("expected error for invalid percent-escape")
	}
}
