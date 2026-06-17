package compression

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"hash/crc32"
	"io"
	"strings"
	"testing"
)

// gunzip is a test helper: decompress a gzip member with the Go standard library, which is an RFC 1952 / RFC 1951 decoder fully independent of Zopfli's encoder.
func gunzip(t *testing.T, b []byte) []byte {
	t.Helper()
	r, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("gzip.NewReader: %v", err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("gzip read: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return out
}

func TestZopfliGzip_RoundTrip(t *testing.T) {
	inputs := []string{
		"",
		"hello world",
		strings.Repeat("the quick brown fox jumps over the lazy dog\n", 200),
	}
	for _, in := range inputs {
		got, err := zopfliGzip([]byte(in), 15)
		if err != nil {
			t.Fatalf("zopfliGzip(%q): %v", truncate(in), err)
		}
		if rt := string(gunzip(t, got)); rt != in {
			t.Errorf("round-trip mismatch for %q: got %q", truncate(in), truncate(rt))
		}
	}
}

func TestZopfliGzip_HeaderIsSpecExact(t *testing.T) {
	// RFC 1952 header per the spec: ID1 ID2 CM FLG | MTIME=0 | XFL=2 OS=255.
	got, err := zopfliGzip([]byte("payload"), 15)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff}
	if !bytes.Equal(got[:10], want) {
		t.Errorf("header = % x, want % x", got[:10], want)
	}
}

func TestZopfliGzip_TrailerIsCRCAndISize(t *testing.T) {
	in := []byte("the quick brown fox")
	got, err := zopfliGzip(in, 15)
	if err != nil {
		t.Fatal(err)
	}
	trailer := got[len(got)-8:]
	if crc := binary.LittleEndian.Uint32(trailer[:4]); crc != crc32.ChecksumIEEE(in) {
		t.Errorf("trailer CRC32 = %#x, want %#x", crc, crc32.ChecksumIEEE(in))
	}
	if size := binary.LittleEndian.Uint32(trailer[4:]); size != uint32(len(in)) {
		t.Errorf("trailer ISIZE = %d, want %d", size, len(in))
	}
}

func TestZopfliGzip_Deterministic(t *testing.T) {
	in := []byte(strings.Repeat("deterministic output is required for plan stability ", 50))
	a, err := zopfliGzip(in, 15)
	if err != nil {
		t.Fatal(err)
	}
	b, err := zopfliGzip(in, 15)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Error("zopfliGzip is not deterministic for identical input + iterations")
	}
}

func TestZopfliGzip_BeatsOrMatchesStdlibGzip(t *testing.T) {
	// Zopfli's whole point: at least as small as the standard library's best gzip on real text. Block-splitting must stay on for this to hold.
	in := []byte(strings.Repeat("aim high in hope and work, a noble logical diagram once recorded never dies\n", 300))
	z, err := zopfliGzip(in, 15)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	w, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if _, err := w.Write(in); err != nil {
		t.Fatal(err)
	}
	w.Close()
	if len(z) > buf.Len() {
		t.Errorf("zopfli (%d bytes) larger than stdlib gzip -9 (%d bytes)", len(z), buf.Len())
	}
}

func truncate(s string) string {
	if len(s) > 40 {
		return s[:40] + "..."
	}
	return s
}
