package compression

import (
	"bytes"
	"compress/gzip"
	"testing"
)

// deterministicText builds n bytes of realistic-entropy text: word-level
// redundancy like real user_data / config, not trivially compressible. An xorshift
// PRNG drives word selection so the corpus is fixed across runs.
func deterministicText(n int) []byte {
	words := []string{
		"the", "server", "config", "instance", "region", "network", "policy",
		"resource", "provider", "terraform", "compute", "storage", "cluster",
		"deploy", "service", "account", "security", "group", "subnet", "gateway",
		"database", "cache", "queue", "worker", "handler", "request", "response",
		"payload", "encode", "decode", "compress", "buffer", "stream", "value",
		"default", "enabled", "timeout", "retries", "backoff", "interval",
		"metadata", "annotation", "label", "selector", "namespace", "container",
		"volume", "mount", "environment", "variable", "secret", "credential",
	}
	var b bytes.Buffer
	seed := uint32(2463534242)
	for b.Len() < n {
		seed ^= seed << 13
		seed ^= seed >> 17
		seed ^= seed << 5
		b.WriteString(words[int(seed>>8)%len(words)])
		b.WriteByte(' ')
	}
	return b.Bytes()[:n]
}

var benchSizes = []struct {
	name string
	n    int
}{
	{"4KB", 4 << 10},
	{"64KB", 64 << 10},
	{"200KB", 200 << 10},
}

// BenchmarkGzipStdlib is the baseline: this is what Terraform's built-in base64gzip
// does (compress/gzip at BestCompression), minus the base64 step.
func BenchmarkGzipStdlib(b *testing.B) {
	for _, s := range benchSizes {
		in := deterministicText(s.n)
		b.Run(s.name, func(b *testing.B) {
			b.SetBytes(int64(len(in)))
			b.ReportAllocs()
			for b.Loop() {
				var buf bytes.Buffer
				w, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
				_, _ = w.Write(in)
				_ = w.Close()
			}
		})
	}
}

func BenchmarkZopfli(b *testing.B) {
	for _, iters := range []int{15} { // 15 is the default
		for _, s := range benchSizes {
			in := deterministicText(s.n)
			b.Run(s.name, func(b *testing.B) {
				b.SetBytes(int64(len(in)))
				b.ReportAllocs()
				for b.Loop() {
					if _, err := zopfliGzip(in, iters); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

func BenchmarkBrotli(b *testing.B) {
	qualities := []int{11, 6, 4} // 11 is the default
	for _, q := range qualities {
		for _, s := range benchSizes {
			in := deterministicText(s.n)
			name := s.name + "/q" + itoa(q)
			b.Run(name, func(b *testing.B) {
				b.SetBytes(int64(len(in)))
				b.ReportAllocs()
				for b.Loop() {
					if _, err := brotliCompress(in, q, brotliDefaultLgwin); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [8]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
