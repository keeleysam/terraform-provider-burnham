package compression

import (
	"bytes"
	"compress/gzip"
	"io"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"
)

// unbrotli is a test helper: decompress a brotli stream with andybalholm's reader.
func unbrotli(t *testing.T, b []byte) []byte {
	t.Helper()
	out, err := io.ReadAll(brotli.NewReader(bytes.NewReader(b)))
	if err != nil {
		t.Fatalf("brotli read: %v", err)
	}
	return out
}

func TestBrotliCompress_RoundTrip(t *testing.T) {
	inputs := []string{
		"",
		"hello world",
		strings.Repeat("the quick brown fox jumps over the lazy dog\n", 200),
	}
	for _, in := range inputs {
		got, err := brotliCompress([]byte(in), 11, 22)
		if err != nil {
			t.Fatalf("brotliCompress(%q): %v", truncate(in), err)
		}
		if rt := string(unbrotli(t, got)); rt != in {
			t.Errorf("round-trip mismatch for %q: got %q", truncate(in), truncate(rt))
		}
	}
}

func TestBrotliCompress_QualityAndWindowMatrix(t *testing.T) {
	in := []byte(strings.Repeat("noble logical diagram\n", 100))
	for _, q := range []int{0, 5, 11} {
		for _, w := range []int{10, 22, 24} {
			got, err := brotliCompress(in, q, w)
			if err != nil {
				t.Fatalf("brotliCompress(q=%d, w=%d): %v", q, w, err)
			}
			if rt := string(unbrotli(t, got)); rt != string(in) {
				t.Errorf("round-trip mismatch at q=%d w=%d", q, w)
			}
		}
	}
}

func TestBrotliCompress_Deterministic(t *testing.T) {
	in := []byte(strings.Repeat("deterministic output is required for plan stability ", 50))
	a, err := brotliCompress(in, 11, 22)
	if err != nil {
		t.Fatal(err)
	}
	b, err := brotliCompress(in, 11, 22)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Error("brotliCompress is not deterministic for identical input + options")
	}
}

func TestBrotliCompress_BeatsGzipOnText(t *testing.T) {
	// Brotli-11's headline claim: smaller than gzip -9 on text-heavy payloads.
	in := []byte(strings.Repeat("aim high in hope and work, a noble logical diagram once recorded never dies\n", 300))
	br, err := brotliCompress(in, 11, 22)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	w, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	w.Write(in)
	w.Close()
	if len(br) >= buf.Len() {
		t.Errorf("brotli-11 (%d bytes) not smaller than gzip -9 (%d bytes)", len(br), buf.Len())
	}
}
