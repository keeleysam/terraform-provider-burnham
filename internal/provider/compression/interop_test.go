package compression

import (
	"bytes"
	"testing"
)

/*
Large-input sanity checks: a ~3 MiB payload exercises buffer and window handling that small fixtures miss. Both decompress in-process — zopfli output through the Go standard library's compress/gzip reader (an RFC 1952 decoder entirely independent of the Zopfli encoder), brotli output through andybalholm's reader — and must reproduce the input exactly.
*/

func TestZopfliGzip_LargeInput(t *testing.T) {
	// iterations=1 keeps the test fast; pass count doesn't affect buffer behavior, only optimization depth.
	in := bytes.Repeat([]byte("abcdefghijklmnopqrstuvwxyz0123456789\n"), 90000)
	gz, err := zopfliGzip(in, 1)
	if err != nil {
		t.Fatal(err)
	}
	if rt := gunzip(t, gz); !bytes.Equal(rt, in) {
		t.Errorf("large-input round-trip mismatch: got %d bytes, want %d", len(rt), len(in))
	}
}

func TestBrotliCompress_LargeInput(t *testing.T) {
	in := bytes.Repeat([]byte("abcdefghijklmnopqrstuvwxyz0123456789\n"), 90000)
	b, err := brotliCompress(in, 5, 22)
	if err != nil {
		t.Fatal(err)
	}
	if rt := unbrotli(t, b); !bytes.Equal(rt, in) {
		t.Errorf("large-input round-trip mismatch: got %d bytes, want %d", len(rt), len(in))
	}
}
