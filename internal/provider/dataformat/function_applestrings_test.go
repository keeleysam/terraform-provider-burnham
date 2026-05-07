package dataformat

import (
	"testing"
)

func TestDecodeAppleStringsBody_Basic(t *testing.T) {
	got, err := decodeAppleStringsBody(`"hello" = "Hello";` + "\n" + `"bye" = "Bye";` + "\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["hello"] != "Hello" || got["bye"] != "Bye" {
		t.Errorf("unexpected: %v", got)
	}
}

func TestDecodeAppleStringsBody_Comments(t *testing.T) {
	src := "// header\n/* multi\nline */\n\"k\" = \"v\"; // trailing\n"
	got, err := decodeAppleStringsBody(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["k"] != "v" {
		t.Errorf("want v, got %q", got["k"])
	}
}

func TestDecodeAppleStringsBody_EscapeSequences(t *testing.T) {
	got, err := decodeAppleStringsBody(`"k" = "a\nb\tc\\d\"e!";` + "\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["k"] != "a\nb\tc\\d\"e!" {
		t.Errorf("unexpected: %q", got["k"])
	}
}

func TestDecodeAppleStringsBody_UTF16LE(t *testing.T) {
	// "k" = "v"; in UTF-16LE with BOM
	src := []byte{0xFF, 0xFE}
	for _, r := range []rune{'"', 'k', '"', ' ', '=', ' ', '"', 'v', '"', ';'} {
		src = append(src, byte(r), 0x00)
	}
	got, err := decodeAppleStringsBody(string(src))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["k"] != "v" {
		t.Errorf("UTF-16LE BOM not handled: %v", got)
	}
}

func TestDecodeAppleStringsBody_UnterminatedString(t *testing.T) {
	_, err := decodeAppleStringsBody(`"k" = "unterminated`)
	if err == nil {
		t.Error("expected error for unterminated string")
	}
}

func TestDecodeAppleStringsBody_MissingSemicolon(t *testing.T) {
	_, err := decodeAppleStringsBody(`"k" = "v"`)
	if err == nil {
		t.Error("expected error for missing semicolon")
	}
}

func TestDecodeAppleStringsBody_Empty(t *testing.T) {
	got, err := decodeAppleStringsBody("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("want empty, got %v", got)
	}
}

func TestEscapeStringsLiteral(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"plain", "plain"},
		{"a\"b", `a\"b`},
		{"a\\b", `a\\b`},
		{"a\nb", `a\nb`},
		{"a\tb", `a\tb`},
	}
	for _, tc := range cases {
		got := escapeAppleStringsLiteral(tc.in)
		if got != tc.want {
			t.Errorf("input %q: want %q, got %q", tc.in, tc.want, got)
		}
	}
}
