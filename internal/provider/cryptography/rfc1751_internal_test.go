package cryptography

import (
	"bytes"
	"encoding/hex"
	"testing"
)

// Vectors are the three worked examples from RFC 1751 itself (§ body):
//   EB33F77EE73D4053                  -> TIDE ITCH SLOW REIN RULE MOT
//   CCAC2AED591056BE4F90FD441C534766  -> RASH BUSH MILK LOOK BAD BRIM AVID GAFF BAIT ROT POD LOVE
//   EFF81F9BFBC65350920CDD7416DE8009  -> TROD MUTE TAIL WARM CHAR KONG HAAG CITY BORE O TEAL AWL

var rfc1751Vectors = []struct {
	hex   string
	words string
}{
	{"EB33F77EE73D4053", "TIDE ITCH SLOW REIN RULE MOT"},
	{"CCAC2AED591056BE4F90FD441C534766", "RASH BUSH MILK LOOK BAD BRIM AVID GAFF BAIT ROT POD LOVE"},
	{"EFF81F9BFBC65350920CDD7416DE8009", "TROD MUTE TAIL WARM CHAR KONG HAAG CITY BORE O TEAL AWL"},
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad test hex %q: %v", s, err)
	}
	return b
}

func TestBtoe_RFCVectors(t *testing.T) {
	for _, v := range rfc1751Vectors {
		got, err := btoeBytes(mustHex(t, v.hex))
		if err != nil {
			t.Fatalf("btoeBytes(%s): unexpected error: %v", v.hex, err)
		}
		if got != v.words {
			t.Errorf("btoeBytes(%s) = %q, want %q", v.hex, got, v.words)
		}
	}
}

func TestEtob_RFCVectors(t *testing.T) {
	for _, v := range rfc1751Vectors {
		got, err := etobWords(v.words)
		if err != nil {
			t.Fatalf("etobWords(%q): unexpected error: %v", v.words, err)
		}
		if want := mustHex(t, v.hex); !bytes.Equal(got, want) {
			t.Errorf("etobWords(%q) = %X, want %s", v.words, got, v.hex)
		}
	}
}

func TestBtoe_AllZero(t *testing.T) {
	// 64 zero bits and zero parity select index 0 ("A") six times.
	got, err := btoeBytes(make([]byte, 8))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "A A A A A A"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRoundTrip(t *testing.T) {
	in := mustHex(t, "0123456789ABCDEFFEDCBA9876543210")
	words, err := btoeBytes(in)
	if err != nil {
		t.Fatalf("btoeBytes: %v", err)
	}
	out, err := etobWords(words)
	if err != nil {
		t.Fatalf("etobWords: %v", err)
	}
	if !bytes.Equal(in, out) {
		t.Errorf("round-trip mismatch: in %X, out %X (via %q)", in, out, words)
	}
}

func TestEtob_CaseInsensitive(t *testing.T) {
	// standard() uppercases input before lookup, so lowercase round-trips.
	got, err := etobWords("tide itch slow rein rule mot")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := mustHex(t, "EB33F77EE73D4053"); !bytes.Equal(got, want) {
		t.Errorf("got %X, want EB33F77EE73D4053", got)
	}
}

func TestEtob_ParityError(t *testing.T) {
	// "A A A A A A" is valid (zero data, zero parity). Replacing the first word
	// with "ABE" (index 1) sets a data bit without updating the parity word, so
	// the parity check must fail.
	if _, err := etobWords("ABE A A A A A"); err == nil {
		t.Fatal("expected parity error, got nil")
	}
}

func TestEtob_UnknownWord(t *testing.T) {
	if _, err := etobWords("ZZZZ ITCH SLOW REIN RULE MOT"); err == nil {
		t.Fatal("expected unknown-word error, got nil")
	}
}

func TestBtoe_BadLength(t *testing.T) {
	if _, err := btoeBytes(make([]byte, 7)); err == nil {
		t.Fatal("expected error for non-multiple-of-8 input, got nil")
	}
	if _, err := btoeBytes(nil); err == nil {
		t.Fatal("expected error for empty input, got nil")
	}
}

func TestEtob_BadWordCount(t *testing.T) {
	if _, err := etobWords("TIDE ITCH SLOW REIN RULE"); err == nil {
		t.Fatal("expected error for non-multiple-of-6 word count, got nil")
	}
}
