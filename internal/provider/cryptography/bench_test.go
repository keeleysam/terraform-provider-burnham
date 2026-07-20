package cryptography

import (
	"testing"
)

// A fixed 32-byte seed; deterministic keygen is the whole point of these functions.
var benchSeed = []byte("burnham-benchmark-seed-0123456789")

func BenchmarkECDSAP256KeyFromSeed(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, err := ecdsaP256KeyFromSeed(benchSeed); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEd25519KeyFromSeed(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, err := ed25519KeyFromSeed(benchSeed[:32]); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkHMAC_SHA256 is the cheap-end reference: a single keyed hash over 1KB.
func BenchmarkHMAC_SHA256(b *testing.B) {
	key := benchSeed
	msg := make([]byte, 1024)
	for i := range msg {
		msg[i] = byte(i)
	}
	sig, err := signJWS(msg, "HS256", key)
	if err != nil {
		b.Fatal(err)
	}
	_ = sig
	b.ReportAllocs()
	for b.Loop() {
		if _, err := signJWS(msg, "HS256", key); err != nil {
			b.Fatal(err)
		}
	}
}
