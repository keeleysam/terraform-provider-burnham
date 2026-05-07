/*
Command genpi computes π to a fixed number of digits and writes the DPD-packed result to pi_packed.bin in the numerics package directory.

Invoked by `go generate ./...`; not part of the production binary.

Layout: IEEE 754-2008 Densely Packed Decimal — three decimal digits in ten bits. The packed bits are written MSB-first, big-endian. So 3,141,592 digits produce ⌈⌈3,141,592 / 3⌉ × 10 / 8⌉ = 1,308,998 bytes.

That's a saving of ~256 KB vs. 4-bit BCD (which would be 1,570,796 bytes at 4 bits/digit). DPD's density (3.33 bits/digit) is within 0.3% of the information-theoretic floor (log₂10 ≈ 3.322 bits/digit). See internal/dpd for the encoding details.

The output path is anchored to this main.go's source location via runtime.Caller, so the tool writes to the right place whether invoked through `go generate ./...` (cwd = directive's directory, anchored is equivalent) or directly via `go run ./internal/provider/numerics/cmd/genpi/` (cwd = wherever, anchored still resolves correctly).
*/

package main

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/keeleysam/terraform-burnham/internal/provider/numerics/internal/chudnovsky"
	"github.com/keeleysam/terraform-burnham/internal/provider/numerics/internal/dpd"
)

// digitCount is the number of decimal digits of π to embed. Must match piEmbeddedDigitCount in the numerics package.
//
// 3,141,592 = ⌊π × 10⁶⌋ — we ship π digits of π. Floored, not rounded: rounding would mean computing one digit of π beyond the cap, and that digit is by definition not in our table, so floor is the only honest choice (cf. RFC 3091's stern warning about returning incorrect digits).
const digitCount = 3_141_592

func main() {
	log.SetFlags(0)
	log.SetPrefix("genpi: ")

	digits := chudnovsky.PiDigits(digitCount)
	if len(digits) != digitCount {
		log.Fatalf("expected %d digits from chudnovsky.PiDigits, got %d", digitCount, len(digits))
	}

	packed := dpdPack(digits)

	dst := outputPath()
	if err := os.WriteFile(dst, packed, 0644); err != nil {
		log.Fatalf("writing %s: %v", dst, err)
	}
	log.Printf("wrote %s (%d bytes for %d digits, DPD-packed)", dst, len(packed), digitCount)
}

// dpdPack packs the ASCII decimal-digit string into DPD format. Triples are written most-significant-bit-first into the output stream; each triple occupies 10 contiguous bits.
//
// The last triple is padded with zero digits if digitCount is not a multiple of 3. We never decode those padding positions (the runtime cap check rejects n past the real digit count).
func dpdPack(digits string) []byte {
	n := len(digits)
	triples := (n + 2) / 3 // ceil(n / 3)
	totalBits := triples * 10
	out := make([]byte, (totalBits+7)/8)

	bitOffset := 0
	for t := 0; t < triples; t++ {
		var d0, d1, d2 byte
		base := t * 3
		d0 = digits[base] - '0'
		if base+1 < n {
			d1 = digits[base+1] - '0'
		}
		if base+2 < n {
			d2 = digits[base+2] - '0'
		}
		code := dpd.Encode(d0, d1, d2)
		writeBits(out, bitOffset, code)
		bitOffset += 10
	}
	return out
}

// writeBits writes the low 10 bits of `value` into `out` starting at the given bit offset, MSB-first within each byte.
//
// Specialized to width=10 and the triple-aligned offset pattern: bitOffset is always t*10 for some integer t, so bitInByte ∈ {0, 2, 4, 6} (never 7). That guarantees the 10-bit code fits within two consecutive bytes — we never need to write a third. The same invariant powers readDPDTriple's matching simplification on the runtime side.
func writeBits(out []byte, bitOffset int, value uint16) {
	byteOffset := bitOffset / 8
	bitInByte := bitOffset % 8 // ∈ {0, 2, 4, 6}

	// Place the 10-bit code left-aligned in a 16-bit window starting at byteOffset, with bitInByte bits of left padding. Shift amount is 16 - 10 - bitInByte = 6 - bitInByte ∈ {0, 2, 4, 6}.
	v := value << uint(6-bitInByte)

	out[byteOffset] |= byte(v >> 8)
	out[byteOffset+1] |= byte(v)
}

// outputPath returns the absolute path to pi_packed.bin in the numerics package, computed relative to *this* source file. Anchoring to source rather than cwd makes the tool robust to being invoked outside `go generate ./...` (where cwd is set to the directive's directory).
func outputPath() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("runtime.Caller(0) failed; cannot locate source file")
	}
	// thisFile = .../numerics/cmd/genpi/main.go
	// target  = .../numerics/pi_packed.bin
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "pi_packed.bin")
}
