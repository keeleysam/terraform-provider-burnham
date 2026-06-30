package encoding

import (
	"bytes"
	"strings"
	"testing"
)

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
